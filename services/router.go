package services

import (
	"time"
	"yao/libs"
	"yao/service"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Method   string
	Function func(*gin.Context)
}

/*
The router table for yaoj back-end server

urls with first letter upper are rpcs, others are apis.
*/
var Router map[string][]Request = map[string][]Request{
	"/GetTime":    {{"POST", GetTime}},
	"/Init":       {{"POST", USInit}},
	"/UserLogin":  {{"POST", service.GinHandler(UserLogin)}},
	"/UserLogout": {{"POST", USLogout}},
	"/Rejudge":    {{"POST", Rejudge}},

	"/user": {
		{"GET", service.GinHandler(UserGet)},
		{"POST", service.GinHandler(UserSignUp)},
		{"PUT", service.GinHandler(UserEdit)},
		{"PATCH", service.GinHandler(UserGrpEdit)},
	},
	"/captcha": {
		{"GET", service.GinHandler(CaptchaGet)},
		{"POST", service.GinHandler(CaptchaPost)},
		{"PATCH", service.GinHandler(CaptchaReload)},
	},
	"/permissions":      {{"GET", PMQuery}, {"POST", PMCreate}, {"PATCH", PMChangeName}, {"DELETE", PMDelete}},
	"/user_permissions": {{"GET", PMQueryUser}, {"POST", PMAddUser}, {"DELETE", PMDeleteUser}},
	"/users":            {{"GET", service.GinHandler(UserList)}},

	"/blog": {
		{"GET", service.GinHandler(BlogGet)},
		{"POST", service.GinHandler(BlogCreate)},
		{"PUT", service.GinHandler(BlogEdit)},
		{"DELETE", service.GinHandler(BlogDel)},
	},
	"/blogs": {{"GET", service.GinHandler(BlogList)}},
	"/likes": {{"POST", service.GinHandler(ClickLike)}},
	"/blog_comments": {
		{"GET", service.GinHandler(BlogCmntGet)},
		{"POST", service.GinHandler(BlogCmntCreate)},
		{"DELETE", service.GinHandler(BlogCmntDel)},
	},
	"/announcements": {
		{"GET", service.GinHandler(AnceGet)},
		{"POST", service.GinHandler(AnceCreate)},
		{"DELETE", service.GinHandler(AnceDel)},
	},
	"/problems": {{"GET", service.GinHandler(ProbList)}},
	"/problem": {
		{"GET", service.GinHandler(ProbGet)},
		{"POST", PRCreate},
		{"PATCH", service.GinHandler(ProbModify)},
	},
	"/problem_permissions": {
		{"GET", service.GinHandler(ProbGetPerm)},
		{"POST", service.GinHandler(ProbAddPerm)},
		{"DELETE", service.GinHandler(ProbDelPerm)},
	},
	"/problem_managers": {
		{"GET", service.GinHandler(ProbGetMgr)},
		{"POST", service.GinHandler(ProbAddMgr)},
		{"DELETE", service.GinHandler(ProbDelMgr)},
	},
	"/problem_data": {
		{"GET", service.GinHandler(ProbDownData)},
		{"PUT", service.GinHandler(ProbPutData)},
	},

	"/contests":             {{"GET", CTList}},
	"/contest":              {{"GET", CTQuery}, {"POST", CTCreate}, {"PATCH", CTModify}},
	"/contest_permissions":  {{"GET", CTGetPermissions}, {"POST", CTAddPermission}, {"DELETE", CTDeletePermission}},
	"/contest_managers":     {{"GET", CTGetManagers}, {"POST", CTAddManager}, {"DELETE", CTDeleteManager}},
	"/contest_participants": {{"GET", CTGetParticipants}, {"POST", CTSignup}, {"DELETE", CTSignout}},
	"/contest_problems":     {{"GET", CTGetProblems}, {"POST", CTAddProblem}, {"DELETE", CTDeleteProblem}},

	"/submissions": {{"GET", SMList}},
	"/submission":  {{"GET", SMQuery}, {"POST", SMSubmit}, {"DELETE", SMDelete}},
	"/custom_test": {{"POST", SMCustomTest}},
}

func GetTime(ctx *gin.Context) {
	libs.RPCWriteBack(ctx, 200, 0, "", map[string]any{"server_time": time.Now()})
}
