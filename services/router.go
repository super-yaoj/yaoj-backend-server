package services

import (
	"net/http"
	"time"
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
	"/GetTime":       {{"POST", service.GinHandler(GetTime)}},
	"/Init":          {{"POST", service.GinHandler(UserInit)}},
	"/UserLogin":     {{"POST", service.GinHandler(UserLogin)}},
	"/UserLogout":    {{"POST", service.GinHandler(UserLogout)}},
	"/Rejudge":       {{"POST", service.GinHandler(Rejudge)}},
	"/FinishContest": {{"POST", service.GinHandler(CtstFinish)}},
	"/judgerlog":     {{"GET", service.GinHandler(JudgerLog)}},

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
	"/permissions": {
		{"GET", service.GinHandler(PermGet)},
		{"POST", service.GinHandler(PermCreate)},
		{"PATCH", service.GinHandler(PermRename)},
		{"DELETE", service.GinHandler(PermDel)},
	},
	"/permission_users": {
		{"GET", service.GinHandler(PermGetUser)},
		{"POST", service.GinHandler(PermAddUser)},
		{"DELETE", service.GinHandler(PermDelUser)},
	},
	"/user_permissions": {
		{"GET", service.GinHandler(UserGetPerm)},
	},
	"/users": {{"GET", service.GinHandler(UserList)}},
	"/ratings": {{"GET", service.GinHandler(UserRating)}},

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
		{"POST", service.GinHandler(ProbAdd)},
		{"PATCH", service.GinHandler(ProbEdit)},
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
	"/problem_statistic": {
		{"GET", service.GinHandler(ProbStatistic)},
	},

	"/contests": {{"GET", service.GinHandler(CtstList)}},
	"/contest": {
		{"GET", service.GinHandler(CtstGet)},
		{"POST", service.GinHandler(CtstCreate)},
		{"PATCH", service.GinHandler(CtstEdit)},
	},
	"/contest_permissions": {
		{"GET", service.GinHandler(CtstPermGet)},
		{"POST", service.GinHandler(CtstPermAdd)},
		{"DELETE", service.GinHandler(CtstPermDel)},
	},
	"/contest_managers": {
		{"GET", service.GinHandler(CtstMgrGet)},
		{"POST", service.GinHandler(CtstMgrAdd)},
		{"DELETE", service.GinHandler(CtstMgrDel)},
	},
	"/contest_participants": {
		{"GET", service.GinHandler(CtstPtcpGet)},
		{"POST", service.GinHandler(CtstSignup)},
		{"DELETE", service.GinHandler(CtstSignout)},
	},
	"/contest_problems": {
		{"GET", service.GinHandler(CtstProbGet)},
		{"POST", service.GinHandler(CtstProbAdd)},
		{"DELETE", service.GinHandler(CtstProbDel)},
	},
	"/contest_standing": {
		{"GET", service.GinHandler(CtstStanding)},
	},
	"/contest_dashboard": {
		{"GET", service.GinHandler(CtstGetDashboard)},
		{"POST", service.GinHandler(CtstAddDashboard)},
	},

	"/submissions": {{"GET", service.GinHandler(SubmList)}},
	"/submission": {
		{"GET", service.GinHandler(SubmGet)},
		{"POST", service.GinHandler(SubmAdd)},
		{"DELETE", service.GinHandler(SubmDel)},
	},
	"/custom_test": {{"POST", service.GinHandler(SubmCustom)}},
}

type GetTimeParam struct {
}

func GetTime(ctx Context, param GetTimeParam) {
	ctx.JSONRPC(http.StatusOK, 0, "", map[string]any{"server_time": time.Now()})
}
