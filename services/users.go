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

const (
	USBanned = 0
	USNormal = 1
	USAdmin  = 2
	USRoot   = 3
)

func checkPU(ctx *gin.Context, name, password string) bool {
	if !internal.ValidPassword(password) || !internal.ValidUsername(name) {
		message := "invalid username"
		if !internal.ValidPassword(password) {
			message = "invalid password"
		}
		libs.RPCWriteBack(ctx, 400, -32600, message, nil)
		return false
	}
	return true
}

type UserSignUpParam struct {
	UserName   string `body:"user_name"`
	Passwd     string `body:"password"`
	Memo       string `body:"remember"`
	VerifyID   string `body:"verify_id"`
	VerifyCode string `body:"verify_code"`
}

func UserSignUp(ctx *gin.Context, param UserSignUpParam) {
	if !checkPU(ctx, param.UserName, param.Passwd) {
		return
	}
	if !VerifyCaptcha(param.VerifyID, param.VerifyCode) {
		libs.APIWriteBack(ctx, 400, "verify code is wrong", nil)
		return
	}
	password := internal.SaltPassword(param.Passwd)
	remember_token := ""
	if param.Memo == "true" {
		remember_token = libs.RandomString(32)
	}
	user_id, err := libs.DBInsertGetId(
		"insert into user_info values (null, ?, ?, \"\", 0, ?, ?, ?, 0, \"\", \"\")",
		param.UserName, password, time.Now(), remember_token, USNormal,
	)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "username has been used by others", nil)
		return
	}
	sess := sessions.Default(ctx)
	sess.Set("user_id", int(user_id))
	sess.Set("user_name", param.UserName)
	sess.Set("user_group", USNormal)
	libs.DBUpdate("insert into user_permissions values (?, ?)", user_id, libs.DefaultGroup)
	libs.DBUpdate("update permissions set count = count + 1 where permission_id=1")
	sess.Save()
	if param.Memo == "true" {
		libs.SetCookie(ctx, "user_id", fmt.Sprint(user_id), true)
		libs.SetCookie(ctx, "remember_token", remember_token, true)
	}
	libs.APIWriteBack(ctx, 200, "", nil)
}

type UserLoginParam struct {
	UserName string `body:"user_name"`
	Passwd   string `body:"password"`
	Memo     string `body:"remember"`
}

func UserLogin(ctx *gin.Context, param UserLoginParam) {
	if !checkPU(ctx, param.UserName, param.Passwd) {
		return
	}
	password := internal.SaltPassword(param.Passwd)
	user := internal.UserSmall{Name: param.UserName}
	err := libs.DBSelectSingle(
		&user, "select user_id, user_group from user_info where user_name=? and password=?",
		param.UserName, password,
	)
	if err != nil {
		libs.RPCWriteBack(ctx, 400, -32600, "username or password is wrong", nil)
		return
	}
	if user.Usergroup == USBanned {
		libs.RPCWriteBack(ctx, 400, -32600, "user is banned", nil)
		return
	}
	sess := sessions.Default(ctx)
	sess.Set("user_id", user.Id)
	sess.Set("user_name", user.Name)
	sess.Set("user_group", user.Usergroup)
	sess.Save()
	if param.Memo == "true" {
		remember_token := libs.RandomString(32)
		libs.SetCookie(ctx, "user_id", fmt.Sprint(user.Id), true)
		libs.SetCookie(ctx, "remember_token", remember_token, true)
		libs.DBUpdate("update user_info set remember_token=? where user_id=?", remember_token, user.Id)
	}
	libs.RPCWriteBack(ctx, 200, 0, "", nil)
}

func USLogout(ctx *gin.Context) {
	libs.DeleteCookie(ctx, "user_id")
	libs.DeleteCookie(ctx, "remember_token")
	sess := sessions.Default(ctx)
	sess.Delete("user_id")
	sess.Delete("user_name")
	sess.Delete("user_group")
	sess.Save()
	libs.RPCWriteBack(ctx, 200, 0, "", nil)
}

func USInit(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	var ret func(internal.UserSmall) = func(user internal.UserSmall) {
		fmt.Println(user)
		if user.Usergroup == USBanned {
			USLogout(ctx)
			libs.RPCWriteBack(ctx, 400, -32600, "user is banned", nil)
			return
		}
		libs.RPCWriteBack(ctx, 200, 0, "", map[string]any{"user_id": user.Id, "user_name": user.Name, "user_group": user.Usergroup, "server_time": time.Now()})
	}

	tmp, err := ctx.Cookie("user_id")
	user := internal.UserSmall{Id: -1, Name: "", Usergroup: USNormal}
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
func ISAdmin(ctx *gin.Context) bool {
	sess := sessions.Default(ctx)
	user_group, err := sess.Get("user_group").(int)
	return err && (user_group == USAdmin || user_group == USRoot)
}

// 从 session 中获取 userid
func GetUserId(ctx *gin.Context) int {
	sess := sessions.Default(ctx)
	user_id, err := sess.Get("user_id").(int)
	if !err {
		return -1
	}
	return user_id
}

type UserGetParam struct {
	UserID int `query:"user_id" binding:"required"`
}

func UserGet(ctx *gin.Context, param UserGetParam) {
	user, err := internal.USQuery(param.UserID)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	}
	user.Password, user.RememberToken = "", ""
	data, err := libs.Struct2Map(user)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", data)
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

func UserEdit(ctx *gin.Context, param UserEditParam) {
	if param.UserID != param.CurUserID {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	password := param.Passwd
	ok, err := internal.CheckPassword(param.UserID, password)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	if !ok {
		libs.APIWriteBack(ctx, 400, "wrong password", nil)
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
		libs.APIWriteBack(ctx, 400, "length of motto or organization is too long", nil)
		return
	}
	if !internal.ValidEmail(email) {
		libs.APIWriteBack(ctx, 400, "invalid email", nil)
		return
	}
	err = internal.USModify(password, param.Gender, motto, email, organization, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
}

type UserGrpEditParam struct {
	UserID  int `body:"user_id" binding:"required"`
	UserGrp int `body:"user_group" binding:"required"`
	CurGrp  int `session:"user_group"`
}

func UserGrpEdit(ctx *gin.Context, param UserGrpEditParam) {
	if !ISAdmin(ctx) || param.CurGrp <= param.UserGrp {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	target, err := internal.USQuerySmall(param.UserGrp)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	} else if target.Usergroup >= param.CurGrp {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err = internal.USGroupEdit(param.UserID, param.UserGrp)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", nil)
	}
}

type UserListParam struct {
	PageSize    int     `query:"pagesize" binding:"required"`
	UserName    *string `query:"user_name"`
	Left        *int    `query:"left"`
	Right       *int    `query:"right"`
	LeftUserID  *int    `query:"left_user_id"`
	RightUserID *int    `query:"right_user_id"`
	LeftRating  *int    `query:"left_rating"`
	RightRating *int    `query:"right_rating"`
}

func UserList(ctx *gin.Context, param UserListParam) {
	if param.PageSize < 1 || param.PageSize > 100 {
		return
	}
	if param.UserName != nil {
		var bound int
		if param.Left != nil {
			bound = *param.Left
		} else if param.Right != nil {
			bound = *param.Right
		} else {
			return
		}
		users, isfull, err := internal.USListByName(*param.UserName+"%", bound, param.PageSize, param.Left != nil)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users, "isfull": isfull})
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
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users, "isfull": isfull})
		}
	}
}
