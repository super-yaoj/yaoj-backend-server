package services

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func validSign(username, password string) error {
	if !internal.ValidPassword(password) {
		return fmt.Errorf("invalid password")
	}
	if !internal.ValidUsername(username) {
		return fmt.Errorf("invalid username")
	}
	return nil
}

// UserSignupParam
type UserSignUpParam struct {
	UserName   string `body:"user_name"`
	Passwd     string `body:"password"`
	Memo       string `body:"remember"`
	VerifyID   string `body:"verify_id"`
	VerifyCode string `body:"verify_code"`
}

func UserSignUp(ctx Context, param UserSignUpParam) {
	if err := validSign(param.UserName, param.Passwd); err != nil {
		ctx.JSONRPC(http.StatusBadRequest, -32600, err.Error(), nil)
		return
	}
	if !VerifyCaptcha(param.VerifyID, param.VerifyCode) {
		ctx.JSONAPI(http.StatusBadRequest, "verify code is wrong", nil)
		return
	}
	password := internal.SaltPassword(param.Passwd)
	remember_token := ""
	if param.Memo == "true" {
		remember_token = libs.RandomString(32)
	}
	user_id, err := libs.DBInsertGetId(
		"insert into user_info values (null, ?, ?, \"\", 0, ?, ?, ?, 0, \"\", \"\")",
		param.UserName, password, time.Now(), remember_token, libs.USNormal,
	)
	if err != nil {
		ctx.JSONAPI(http.StatusBadRequest, "username has been used by others", nil)
		return
	}
	sess := sessions.Default(ctx.Context)
	sess.Set("user_id", int(user_id))
	sess.Set("user_name", param.UserName)
	sess.Set("user_group", libs.USNormal)
	libs.DBUpdate("insert into user_permissions values (?, ?)", user_id, libs.DefaultGroup)
	libs.DBUpdate("update permissions set count = count + 1 where permission_id=1")
	sess.Save()
	if param.Memo == "true" {
		libs.SetCookie(ctx.Context, "user_id", fmt.Sprint(user_id), true)
		libs.SetCookie(ctx.Context, "remember_token", remember_token, true)
	}
	ctx.JSONAPI(http.StatusOK, "", nil)
}

type UserLoginParam struct {
	UserName string `body:"user_name"`
	Passwd   string `body:"password"`
	Memo     string `body:"remember"`
}

func UserLogin(ctx Context, param UserLoginParam) {
	if err := validSign(param.UserName, param.Passwd); err != nil {
		ctx.JSONRPC(http.StatusBadRequest, -32600, err.Error(), nil)
		return
	}
	password := internal.SaltPassword(param.Passwd)
	user := internal.UserBase{Name: param.UserName}
	err := libs.DBSelectSingle(
		&user, "select user_id, user_group from user_info where user_name=? and password=?",
		param.UserName, password,
	)
	if err != nil {
		ctx.JSONRPC(http.StatusBadRequest, -32600, "username or password is wrong", nil)
		return
	}
	if user.Usergroup == libs.USBanned {
		ctx.JSONRPC(http.StatusBadRequest, -32600, "user is banned", nil)
		return
	}
	sess := sessions.Default(ctx.Context)
	sess.Set("user_id", user.Id)
	sess.Set("user_name", user.Name)
	sess.Set("user_group", user.Usergroup)
	sess.Save()
	if param.Memo == "true" {
		remember_token := libs.RandomString(32)
		libs.SetCookie(ctx.Context, "user_id", fmt.Sprint(user.Id), true)
		libs.SetCookie(ctx.Context, "remember_token", remember_token, true)
		libs.DBUpdate("update user_info set remember_token=? where user_id=?", remember_token, user.Id)
	}
	ctx.JSONRPC(http.StatusOK, 0, "", nil)
}

type UserLogoutParam struct {
}

func UserLogout(ctx Context, param UserLogoutParam) {
	libs.DeleteCookie(ctx.Context, "user_id")
	libs.DeleteCookie(ctx.Context, "remember_token")
	sess := sessions.Default(ctx.Context)
	sess.Delete("user_id")
	sess.Delete("user_name")
	sess.Delete("user_group")
	sess.Save()
	ctx.JSONRPC(http.StatusOK, 0, "", nil)
}

type UserInitParam struct {
}

func UserInit(ctx Context, param UserInitParam) {
	sess := sessions.Default(ctx.Context)
	ret := func(user internal.UserBase) {
		fmt.Println(user)
		if user.Usergroup == libs.USBanned {
			UserLogout(ctx, UserLogoutParam{})
			ctx.JSONRPC(http.StatusBadRequest, -32600, "user is banned", nil)
			return
		}
		ctx.JSONRPC(http.StatusOK, 0, "", gin.H{
			"user_id":     user.Id,
			"user_name":   user.Name,
			"user_group":  user.Usergroup,
			"server_time": time.Now(),
			"is_admin":    libs.IsAdmin(user.Usergroup),
		})
	}

	tmp, err := ctx.Cookie("user_id")
	user := internal.UserBase{Id: -1, Name: "", Usergroup: libs.USNormal}
	if err == nil {
		id, err := strconv.Atoi(tmp)
		remember_token, err1 := ctx.Cookie("remember_token")
		if err == nil && err1 == nil {
			err = libs.DBSelectSingle(&user, "select user_id, user_name, user_group from user_info where user_id=? and remember_token=?", id, remember_token)
			if err == nil {
				sess.Set("user_id", id)
				sess.Set("user_name", user.Name)
				sess.Set("user_group", user.Usergroup)
				sess.Save()
				ret(user)
				return
			}
		}
	}
	tmp1, err1 := sess.Get("user_id").(int)
	if err1 {
		user.Id = tmp1
		user.Name = sess.Get("user_name").(string)
		user.Usergroup = sess.Get("user_group").(int)
	}
	ret(user)
}

type UserGetParam struct {
	UserID int `query:"user_id" validate:"required,userid"`
}

func UserGet(ctx Context, param UserGetParam) {
	user, err := internal.USQuery(param.UserID)
	user.Password, user.RememberToken = "", ""
	data, err := libs.Struct2Map(user)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", data)
	}
}

type UserEditParam struct {
	Auth
	TargetID  int    `body:"user_id" validate:"required,userid"`
	Gender    int    `body:"gender" validate:"gte=0,lte=2"`
	Passwd    string `body:"password"`
	NewPasswd string `body:"new_password"`
	Motto     string `body:"motto" validate:"lte=350"`
	Email     string `body:"email" validate:"email,lte=70"`
	Org       string `body:"organization" validate:"lte=350"`
}

func UserEdit(ctx Context, param UserEditParam) {
	param.NewPermit().Try(func() (any, bool) {
		//only user himself can modify profiles
		return nil, param.TargetID == param.UserID
	}).Success(func(a any) {
		password := param.Passwd
		ok, err := internal.CheckPassword(param.UserID, password)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
		if !ok {
			ctx.JSONAPI(http.StatusBadRequest, "wrong password", nil)
			return
		}
		new_password := param.NewPasswd
		if new_password != "" && internal.ValidPassword(new_password) {
			password = new_password
		}
		password = internal.SaltPassword(password)
		motto, email, organization := param.Motto, param.Email, param.Org
		err = internal.USModify(password, param.Gender, motto, email, organization, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
	}).FailAPIStatusForbidden(ctx)
}

type UserGrpEditParam struct {
	Auth
	TargetID  int `body:"user_id" validate:"required,userid"`
	TargetGrp int `body:"user_group" validate:"required"`
}

func UserGrpEdit(ctx Context, param UserGrpEditParam) {
	param.NewPermit().AsAdmin().Try(func() (any, bool) {
		//user cannot modify others' permissions which are higher or equal than himself
		return nil, param.UserGrp > param.TargetGrp
	}).Try(func() (any, bool) {
		target, _ := internal.UserQueryBase(param.UserID)
		//users cannot modify others which have permissions higher or equal than himself
		return target, param.UserGrp > target.Usergroup
	}).Success(func(a any) {
		err := internal.USGroupEdit(param.UserID, param.TargetGrp)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type UserListParam struct {
	UserName *string `query:"user_name"`
	Page     `validate:"pagecanbound"`
	LeftID   *int `query:"left_user_id"`
	RightID  *int `query:"right_user_id"`
}

func UserList(ctx Context, param UserListParam) {
	if param.UserName != nil {
		users, isfull, err := internal.USListByName(*param.UserName+"%", param.Bound(), *param.PageSize, param.IsLeft())
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users, "isfull": isfull})
		}
	} else {
		bound_user_id := libs.If(param.IsLeft(), param.LeftID, param.RightID)
		if bound_user_id == nil {
			ctx.JSONAPI(http.StatusBadRequest, "", nil)
			return
		}
		users, isfull, err := internal.USList(*bound_user_id, param.Bound(), *param.PageSize, param.LeftID != nil)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users, "isfull": isfull})
		}
	}
}

type UserRatingParam struct {
	UserId int `query:"user_id" validate:"required,userid"`
}

func UserRating(ctx Context, param UserRatingParam) {
	var ratings []struct {
		Rating    int       `db:"rating" json:"rating"`
		ContestId int       `db:"contest_id" json:"contest_id"`
		Time      time.Time `db:"time" json:"time"`
		Title     string    `db:"title" json:"title"`
	}
	err := libs.DBSelectAll(&ratings, "select rating, a.contest_id, title, time from ((select rating, contest_id, time from ratings where user_id=?) as a inner join contests on a.contest_id=contests.contest_id)", param.UserId)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"ratings": ratings})
	}
}

type UserGetPermParam struct {
	Auth
	PermUserID int `query:"user_id" validate:"required,userid"`
}

func UserGetPerm(ctx Context, param UserGetPermParam) {
	param.NewPermit().Try(func() (any, bool) {
		return nil, param.IsAdmin() || param.PermUserID == param.UserID
	}).Success(func(a any) {
		permissions, err := internal.USQueryPermission(param.PermUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": permissions})
		}
	}).FailAPIStatusForbidden(ctx)
}
