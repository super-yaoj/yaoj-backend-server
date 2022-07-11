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

func USSignup(ctx *gin.Context) {
	name := ctx.PostForm("user_name")
	password := ctx.PostForm("password")
	remember := ctx.PostForm("remember")
	if !checkPU(ctx, name, password) {
		return
	}
	verify_id, verify_code := ctx.PostForm("verify_id"), ctx.PostForm("verify_code")
	if !VerifyCaptcha(verify_id, verify_code) {
		libs.APIWriteBack(ctx, 400, "verify code is wrong", nil)
		return
	}
	password = internal.SaltPassword(password)
	remember_token := ""
	if remember == "true" {
		remember_token = libs.RandomString(32)
	}
	user_id, err := libs.DBInsertGetId("insert into user_info values (null, ?, ?, \"\", 0, ?, ?, ?, 0, \"\", \"\")", name, password, time.Now(), remember_token, USNormal)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "username has been used by others", nil)
		return
	}
	sess := sessions.Default(ctx)
	sess.Set("user_id", int(user_id))
	sess.Set("user_name", name)
	sess.Set("user_group", USNormal)
	libs.DBUpdate("insert into user_permissions values (?, ?)", user_id, libs.DefaultGroup)
	libs.DBUpdate("update permissions set count = count + 1 where permission_id=1")
	sess.Save()
	if remember == "true" {
		libs.SetCookie(ctx, "user_id", fmt.Sprint(user_id), true)
		libs.SetCookie(ctx, "remember_token", remember_token, true)
	}
	libs.APIWriteBack(ctx, 200, "", nil)
}

func USLogin(ctx *gin.Context) {
	name := ctx.PostForm("user_name")
	password := ctx.PostForm("password")
	remember := ctx.PostForm("remember")
	if !checkPU(ctx, name, password) {
		return
	}
	password = internal.SaltPassword(password)
	user := internal.UserSmall{Name: name}
	err := libs.DBSelectSingle(&user, "select user_id, user_group from user_info where user_name=? and password=?", name, password)
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
	if remember == "true" {
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

func USQuery(ctx *gin.Context) {
	id, ok := libs.GetInt(ctx, "user_id")
	if !ok {
		return
	}
	user, err := internal.USQuery(id)
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

func USModify(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	cur, ok := sess.Get("user_id").(int)
	user_id, ok1 := libs.PostInt(ctx, "user_id")
	if !ok1 {
		return
	}
	if !ok || user_id != cur {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	password := ctx.PostForm("password")
	ok, err := internal.CheckPassword(user_id, password)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	if !ok {
		libs.APIWriteBack(ctx, 400, "wrong password", nil)
		return
	}
	new_password := ctx.PostForm("new_password")
	if new_password != "" && internal.ValidPassword(new_password) {
		password = new_password
	}
	password = internal.SaltPassword(password)
	gender, ok := libs.PostIntRange(ctx, "gender", 0, 2)
	if !ok {
		return
	}
	motto, email, organization := ctx.PostForm("motto"), ctx.PostForm("email"), ctx.PostForm("organization")
	if len(motto) > 350 || len(organization) > 150 {
		libs.APIWriteBack(ctx, 400, "length of motto or organization is too long", nil)
		return
	}
	if !internal.ValidEmail(email) {
		libs.APIWriteBack(ctx, 400, "invalid email", nil)
		return
	}
	err = internal.USModify(password, gender, motto, email, organization, user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
}

func USGroupEdit(ctx *gin.Context) {
	user_id, ok := libs.PostInt(ctx, "user_id")
	if !ok {
		return
	}
	user_group, ok := libs.PostIntRange(ctx, "user_group", USBanned, USAdmin)
	if !ok {
		return
	}
	cur_group, ok := sessions.Default(ctx).Get("user_group").(int)
	if !ok || !ISAdmin(ctx) || cur_group <= user_group {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	target, err := internal.USQuerySmall(user_id)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	} else if target.Usergroup >= cur_group {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err = internal.USGroupEdit(user_id, user_group)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", nil)
	}
}

func USList(ctx *gin.Context) {
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok {
		return
	}
	user_name, searchname := ctx.GetQuery("user_name")
	if searchname {
		_, isleft := ctx.GetQuery("left")
		bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
		if !ok {
			return
		}
		users, isfull, err := internal.USListByName(user_name+"%", bound, pagesize, isleft)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users, "isfull": isfull})
		}
	} else {
		_, isleft := ctx.GetQuery("left_user_id")
		bound_user_id, ok := libs.GetInt(ctx, libs.If(isleft, "left_user_id", "right_user_id"))
		if !ok {
			return
		}
		bound_rating, ok := libs.GetInt(ctx, libs.If(isleft, "left_rating", "right_user_id"))
		if !ok {
			return
		}
		users, isfull, err := internal.USList(bound_user_id, bound_rating, pagesize, isleft)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users, "isfull": isfull})
		}
	}
}
