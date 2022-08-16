package services

import (
	"net/http"
	"time"
	"yao/service"
)

type RestApi = service.RestApi

/*
The router table for yaoj back-end server

urls with first letter upper are rpcs, others are apis.
*/
var Router = map[string]RestApi{
	"/GetTime":       {"POST": service.GeneralHandler(GetTime)},
	"/Init":          {"POST": service.GeneralHandler(UserInit)},
	"/UserLogin":     {"POST": service.GeneralHandler(UserLogin)},
	"/UserLogout":    {"POST": service.GeneralHandler(UserLogout)},
	"/Rejudge":       {"POST": service.GeneralHandler(Rejudge)},
	"/FinishContest": {"POST": service.GeneralHandler(CtstFinish)},
	"/judgerlog":     {"GET": service.GeneralHandler(JudgerLog)},

	"/user": {
		"GET":   service.GeneralHandler(UserGet),
		"POST":  service.GeneralHandler(UserSignUp),
		"PUT":   service.GeneralHandler(UserEdit),
		"PATCH": service.GeneralHandler(UserGrpEdit),
	},
	"/captcha": {
		"GET":   service.GeneralHandler(CaptchaGet),
		"POST":  service.GeneralHandler(CaptchaPost),
		"PATCH": service.GeneralHandler(CaptchaReload),
	},
	"/permissions": {
		"GET":    service.GeneralHandler(PermGet),
		"POST":   service.GeneralHandler(PermCreate),
		"PATCH":  service.GeneralHandler(PermRename),
		"DELETE": service.GeneralHandler(PermDel),
	},
	"/permission_users": {
		"GET":    service.GeneralHandler(PermGetUser),
		"POST":   service.GeneralHandler(PermAddUser),
		"DELETE": service.GeneralHandler(PermDelUser),
	},
	"/user_permissions": {
		"GET": service.GeneralHandler(UserGetPerm),
	},
	"/users":   {"GET": service.GeneralHandler(UserList)},
	"/ratings": {"GET": service.GeneralHandler(UserRating)},

	"/blog": {
		"GET":    service.GeneralHandler(BlogGet),
		"POST":   service.GeneralHandler(BlogCreate),
		"PUT":    service.GeneralHandler(BlogEdit),
		"DELETE": service.GeneralHandler(BlogDel),
	},
	"/blogs": {"GET": service.GeneralHandler(BlogList)},
	"/likes": {"POST": service.GeneralHandler(ClickLike)},
	"/blog_comments": {
		"GET":    service.GeneralHandler(BlogCmntGet),
		"POST":   service.GeneralHandler(BlogCmntCreate),
		"DELETE": service.GeneralHandler(BlogCmntDel),
	},
	"/announcements": {
		"GET":    service.GeneralHandler(AnceGet),
		"POST":   service.GeneralHandler(AnceCreate),
		"DELETE": service.GeneralHandler(AnceDel),
	},

	"/problems": {"GET": service.GeneralHandler(ProbList)},
	"/problem": {
		"GET":   service.GeneralHandler(ProbGet),
		"POST":  service.GeneralHandler(ProbAdd),
		"PATCH": service.GeneralHandler(ProbEdit),
	},
	"/problem_permissions": {
		"GET":    service.GeneralHandler(ProbGetPerm),
		"POST":   service.GeneralHandler(ProbAddPerm),
		"DELETE": service.GeneralHandler(ProbDelPerm),
	},
	"/problem_managers": {
		"GET":    service.GeneralHandler(ProbGetMgr),
		"POST":   service.GeneralHandler(ProbAddMgr),
		"DELETE": service.GeneralHandler(ProbDelMgr),
	},
	"/problem_data": {
		"GET": service.GeneralHandler(ProbDownData),
		"PUT": service.GeneralHandler(ProbPutData),
	},
	"/problem_statistic": {
		"GET": service.GeneralHandler(ProbStatistic),
	},

	"/contests": {"GET": service.GeneralHandler(CtstList)},
	"/contest": {
		"GET":   service.GeneralHandler(CtstGet),
		"POST":  service.GeneralHandler(CtstCreate),
		"PATCH": service.GeneralHandler(CtstEdit),
	},
	"/contest_permissions": {
		"GET":    service.GeneralHandler(CtstPermGet),
		"POST":   service.GeneralHandler(CtstPermAdd),
		"DELETE": service.GeneralHandler(CtstPermDel),
	},
	"/contest_managers": {
		"GET":    service.GeneralHandler(CtstMgrGet),
		"POST":   service.GeneralHandler(CtstMgrAdd),
		"DELETE": service.GeneralHandler(CtstMgrDel),
	},
	"/contest_participants": {
		"GET":    service.GeneralHandler(CtstPtcpGet),
		"POST":   service.GeneralHandler(CtstSignup),
		"DELETE": service.GeneralHandler(CtstSignout),
	},
	"/contest_problems": {
		"GET":    service.GeneralHandler(CtstProbGet),
		"POST":   service.GeneralHandler(CtstProbAdd),
		"DELETE": service.GeneralHandler(CtstProbDel),
	},
	"/contest_standing": {
		"GET": service.GeneralHandler(CtstStanding),
	},
	"/contest_dashboard": {
		"GET":  service.GeneralHandler(CtstGetDashboard),
		"POST": service.GeneralHandler(CtstAddDashboard),
	},

	"/submissions": {"GET": service.GeneralHandler(SubmList)},
	"/submission": {
		"GET":    service.GeneralHandler(SubmGet),
		"POST":   service.GeneralHandler(SubmAdd),
		"DELETE": service.GeneralHandler(SubmDel),
	},
	"/custom_test": {"POST": service.GeneralHandler(SubmCustom)},
}

type GetTimeParam struct {
}

func GetTime(ctx *Context, param GetTimeParam) {
	ctx.JSONRPC(http.StatusOK, 0, "", map[string]any{"server_time": time.Now()})
}
