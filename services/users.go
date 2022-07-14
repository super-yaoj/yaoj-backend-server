package services

import (
	"fmt"
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

type UserSignUpParam struct {
	UserName   string `body:"user_name"`
	Passwd     string `body:"password"`
	Memo       string `body:"remember"`
	VerifyID   string `body:"verify_id"`
	VerifyCode string `body:"verify_code"`
}

func UserSignUp(ctx Context, param UserSignUpParam) {
	if err := validSign(param.UserName, param.Passwd); err != nil {
		ctx.JSONRPC(400, -32600, err.Error(), nil)
		return
	}
	if !VerifyCaptcha(param.VerifyID, param.VerifyCode) {
		ctx.JSONAPI(400, "verify code is wrong", nil)
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
		ctx.JSONAPI(400, "username has been used by others", nil)
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
	ctx.JSONAPI(200, "", nil)
}

type UserLoginParam struct {
	UserName string `body:"user_name"`
	Passwd   string `body:"password"`
	Memo     string `body:"remember"`
}

func UserLogin(ctx Context, param UserLoginParam) {
	if err := validSign(param.UserName, param.Passwd); err != nil {
		ctx.JSONRPC(400, -32600, err.Error(), nil)
		return
	}
	password := internal.SaltPassword(param.Passwd)
	user := internal.UserSmall{Name: param.UserName}
	err := libs.DBSelectSingle(
		&user, "select user_id, user_group from user_info where user_name=? and password=?",
		param.UserName, password,
	)
	if err != nil {
		ctx.JSONRPC(400, -32600, "username or password is wrong", nil)
		return
	}
	if user.Usergroup == libs.USBanned {
		ctx.JSONRPC(400, -32600, "user is banned", nil)
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
	ctx.JSONRPC(200, 0, "", nil)
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
	ctx.JSONRPC(200, 0, "", nil)
}

type UserInitParam struct {
}

func UserInit(ctx Context, param UserInitParam) {
	sess := sessions.Default(ctx.Context)
	var ret func(internal.UserSmall) = func(user internal.UserSmall) {
		fmt.Println(user)
		if user.Usergroup == libs.USBanned {
			UserLogout(ctx, UserLogoutParam{})
			ctx.JSONRPC(400, -32600, "user is banned", nil)
			return
		}
		ctx.JSONRPC(200, 0, "", gin.H{
			"user_id":     user.Id,
			"user_name":   user.Name,
			"user_group":  user.Usergroup,
			"server_time": time.Now(),
			"is_admin":    libs.IsAdmin(user.Usergroup),
		})
	}

	tmp, err := ctx.Cookie("user_id")
	user := internal.UserSmall{Id: -1, Name: "", Usergroup: libs.USNormal}
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

// 根据 session 中的 user_group 判断是否是管理
// func ISAdmin(ctx Context) bool {
// 	sess := sessions.Default(ctx)
// 	user_group, ok := sess.Get("user_group").(int)
// 	return ok && libs.IsAdmin(user_group)
// }

// 从 session 中获取 userid
// func GetUserId(ctx Context) int {
// 	sess := sessions.Default(ctx)
// 	user_id, err := sess.Get("user_id").(int)
// 	if !err {
// 		return -1
// 	}
// 	return user_id
// }

type UserGetParam struct {
	UserID int `query:"user_id" binding:"required"`
}

func UserGet(ctx Context, param UserGetParam) {
	user, err := internal.USQuery(param.UserID)
	if err != nil {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	}
	user.Password, user.RememberToken = "", ""
	data, err := libs.Struct2Map(user)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", data)
	}
}

type UserEditParam struct {
	CurUserID int    `session:"user_id"`
	UserID    int    `body:"user_id" binding:"required"`
	Gender    int    `body:"gender" binding:"required"`
	Passwd    string `body:"password"`
	NewPasswd string `body:"new_password"`
	Motto     string `body:"motto"`
	Email     string `body:"email"`
	Org       string `body:"organization"`
}

func UserEdit(ctx Context, param UserEditParam) {
	if param.UserID != param.CurUserID {
		ctx.JSONAPI(403, "", nil)
		return
	}
	password := param.Passwd
	ok, err := internal.CheckPassword(param.UserID, password)
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
	if !ok {
		ctx.JSONAPI(400, "wrong password", nil)
		return
	}
	new_password := param.NewPasswd
	if new_password != "" && internal.ValidPassword(new_password) {
		password = new_password
	}
	password = internal.SaltPassword(password)
	if param.Gender < 0 || param.Gender > 2 {
		return
	}
	motto, email, organization := param.Motto, param.Email, param.Org
	if len(motto) > 350 || len(organization) > 150 {
		ctx.JSONAPI(400, "length of motto or organization is too long", nil)
		return
	}
	if !internal.ValidEmail(email) {
		ctx.JSONAPI(400, "invalid email", nil)
		return
	}
	err = internal.USModify(password, param.Gender, motto, email, organization, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
}

type UserGrpEditParam struct {
	UserID  int `body:"user_id" binding:"required"`
	UserGrp int `body:"user_group" binding:"required"`
	CurGrp  int `session:"user_group" validate:"admin"`
}

func UserGrpEdit(ctx Context, param UserGrpEditParam) {
	if param.CurGrp <= param.UserGrp {
		ctx.JSONAPI(403, "", nil)
		return
	}
	target, err := internal.USQuerySmall(param.UserGrp)
	if err != nil {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	} else if target.Usergroup >= param.CurGrp {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err = internal.USGroupEdit(param.UserID, param.UserGrp)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", nil)
	}
}

type UserListParam struct {
	UserName *string `query:"user_name"`
	Page
	LeftUserID  *int `query:"left_user_id"`
	RightUserID *int `query:"right_user_id"`
	LeftRating  *int `query:"left_rating"`
	RightRating *int `query:"right_rating"`
}

func UserList(ctx Context, param UserListParam) {
	if param.UserName != nil {
		if !param.CanBound() {
			return
		}
		users, isfull, err := internal.USListByName(*param.UserName+"%", param.Bound(), param.PageSize, param.IsLeft())
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(200, "", map[string]any{"data": users, "isfull": isfull})
		}
	} else {
		var bound_user_id, bound_rating int
		if param.LeftUserID != nil {
			bound_user_id = *param.LeftUserID
			bound_rating = *param.LeftRating
		} else if param.RightUserID != nil {
			bound_user_id = *param.RightUserID
			bound_rating = *param.RightRating
		} else {
			return
		}
		users, isfull, err := internal.USList(bound_user_id, bound_rating, param.PageSize, param.LeftUserID != nil)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(200, "", map[string]any{"data": users, "isfull": isfull})
		}
	}
}
