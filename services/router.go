package services

import (
	"net/http"
	"time"
	"yao/server"
)

type RestApi = server.RestApi

/*
The router table for yaoj back-end server

urls with first letter upper are rpcs, others are apis.
*/
var Router = map[string]RestApi{
	"/GetTime":       {"POST": server.GeneralHandler(GetTime)},
	"/Init":          {"POST": server.GeneralHandler(UserInit)},
	"/UserLogin":     {"POST": server.GeneralHandler(UserLogin)},
	"/UserLogout":    {"POST": server.GeneralHandler(UserLogout)},
	"/Rejudge":       {"POST": server.GeneralHandler(Rejudge)},
	"/FinishContest": {"POST": server.GeneralHandler(CtstFinish)},
	"/judgerlog":     {"GET": server.GeneralHandler(JudgerLog)},

	"/user": {
		"GET":   server.GeneralHandler(UserGet),
		"POST":  server.GeneralHandler(UserSignUp),
		"PUT":   server.GeneralHandler(UserEdit),
		"PATCH": server.GeneralHandler(UserGrpEdit),
	},
	"/captcha": {
		"GET":   server.GeneralHandler(CaptchaGet),
		"POST":  server.GeneralHandler(CaptchaPost),
		"PATCH": server.GeneralHandler(CaptchaReload),
	},
	"/permissions": {
		"GET":    server.GeneralHandler(PermGet),
		"POST":   server.GeneralHandler(PermCreate),
		"PATCH":  server.GeneralHandler(PermRename),
		"DELETE": server.GeneralHandler(PermDel),
	},
	"/permission_users": {
		"GET":    server.GeneralHandler(PermGetUser),
		"POST":   server.GeneralHandler(PermAddUser),
		"DELETE": server.GeneralHandler(PermDelUser),
	},
	"/user_permissions": {
		"GET": server.GeneralHandler(UserGetPerm),
	},
	"/users":   {"GET": server.GeneralHandler(UserList)},
	"/ratings": {"GET": server.GeneralHandler(UserRating)},

	"/blog": {
		"GET":    server.GeneralHandler(BlogGet),
		"POST":   server.GeneralHandler(BlogCreate),
		"PUT":    server.GeneralHandler(BlogEdit),
		"DELETE": server.GeneralHandler(BlogDel),
	},
	"/blogs": {"GET": server.GeneralHandler(BlogList)},
	"/likes": {"POST": server.GeneralHandler(ClickLike)},
	"/blog_comments": {
		"GET":    server.GeneralHandler(BlogCmntGet),
		"POST":   server.GeneralHandler(BlogCmntCreate),
		"DELETE": server.GeneralHandler(BlogCmntDel),
	},
	"/announcements": {
		"GET":    server.GeneralHandler(AnceGet),
		"POST":   server.GeneralHandler(AnceCreate),
		"DELETE": server.GeneralHandler(AnceDel),
	},

	"/problems": {"GET": server.GeneralHandler(ProbList)},
	"/problem": {
		"GET":   server.GeneralHandler(ProbGet),
		"POST":  server.GeneralHandler(ProbAdd),
		"PATCH": server.GeneralHandler(ProbEdit),
	},
	"/problem_permissions": {
		"GET":    server.GeneralHandler(ProbGetPerm),
		"POST":   server.GeneralHandler(ProbAddPerm),
		"DELETE": server.GeneralHandler(ProbDelPerm),
	},
	"/problem_managers": {
		"GET":    server.GeneralHandler(ProbGetMgr),
		"POST":   server.GeneralHandler(ProbAddMgr),
		"DELETE": server.GeneralHandler(ProbDelMgr),
	},
	"/problem_data": {
		"GET": server.GeneralHandler(ProbDownData),
		"PUT": server.GeneralHandler(ProbPutData),
	},
	"/problem_statistic": {
		"GET": server.GeneralHandler(ProbStatistic),
	},

	"/contests": {"GET": server.GeneralHandler(CtstList)},
	"/contest": {
		"GET":   server.GeneralHandler(CtstGet),
		"POST":  server.GeneralHandler(CtstCreate),
		"PATCH": server.GeneralHandler(CtstEdit),
	},
	"/contest_permissions": {
		"GET":    server.GeneralHandler(CtstPermGet),
		"POST":   server.GeneralHandler(CtstPermAdd),
		"DELETE": server.GeneralHandler(CtstPermDel),
	},
	"/contest_managers": {
		"GET":    server.GeneralHandler(CtstMgrGet),
		"POST":   server.GeneralHandler(CtstMgrAdd),
		"DELETE": server.GeneralHandler(CtstMgrDel),
	},
	"/contest_participants": {
		"GET":    server.GeneralHandler(CtstPtcpGet),
		"POST":   server.GeneralHandler(CtstSignup),
		"DELETE": server.GeneralHandler(CtstSignout),
	},
	"/contest_problems": {
		"GET":    server.GeneralHandler(CtstProbGet),
		"POST":   server.GeneralHandler(CtstProbAdd),
		"DELETE": server.GeneralHandler(CtstProbDel),
	},
	"/contest_standing": {
		"GET": server.GeneralHandler(CtstStanding),
	},
	"/contest_dashboard": {
		"GET":  server.GeneralHandler(CtstGetDashboard),
		"POST": server.GeneralHandler(CtstAddDashboard),
	},

	"/submissions": {"GET": server.GeneralHandler(SubmList)},
	"/submission": {
		"GET":    server.GeneralHandler(SubmGet),
		"POST":   server.GeneralHandler(SubmAdd),
		"DELETE": server.GeneralHandler(SubmDel),
	},
	"/custom_test": {"POST": server.GeneralHandler(SubmCustom)},
}

type GetTimeParam struct {
}

func GetTime(ctx *Context, param GetTimeParam) {
	ctx.JSONRPC(http.StatusOK, 0, "", map[string]any{"server_time": time.Now()})
}
