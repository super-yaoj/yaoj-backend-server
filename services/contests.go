package services

import (
	"net/http"
	"strings"
	"time"
	"yao/internal"
	"yao/libs"
)

type CtstListParam struct {
	Page
	Auth
}

func CtstList(ctx Context, param CtstListParam) {
	if !param.Page.CanBound() {
		return
	}
	contests, isfull, err := internal.CTList(
		param.Page.Bound(), param.PageSize, param.UserID, param.IsLeft(), libs.IsAdmin(param.UserGrp),
	)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"isfull": isfull, "data": contests})
	}
}

type CtstGetParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstGet(ctx Context, param CtstGetParam) {
	contest, _ := internal.CTQuery(param.CtstID, param.UserID)
	can_edit := param.CanEditCtst(param.CtstID)
	if !param.CanEnterCtst(contest, param.CanEditCtst(contest.Id)) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	ctx.JSONAPI(http.StatusOK, "", map[string]any{"contest": contest, "can_edit": can_edit})
}

type CtstProbGetParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstProbGet(ctx Context, param CtstProbGetParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if !param.CanEnterCtst(contest, param.CanEditCtst(contest.Id)) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	problems, err := internal.CTGetProblems(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": problems})
	}
}

type CtstProbAddParam struct {
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
	ProbID int `body:"problem_id" binding:"required" validate:"probid"`
	Auth
}

func CtstProbAdd(ctx Context, param CtstProbAddParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err := internal.CTAddProblem(param.CtstID, param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstProbDelParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
	Auth
}

func CtstProbDel(ctx Context, param CtstProbDelParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err := internal.CTDeleteProblem(param.CtstID, param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstCreateParam struct {
	UserGrp int `session:"user_group" validate:"admin"`
}

func CtstCreate(ctx Context, param CtstCreateParam) {
	id, err := internal.CTCreate()
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
	}
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Greater_Than
// gte,lte 用在 string 上表示长度限制
type CtstEditParam struct {
	CtstID       int    `body:"contest_id" binding:"required" validate:"ctstid"`
	Title        string `body:"title" validate:"gte=0,lte=190"`
	StartTime    string `body:"start_time"`
	Duration     int    `body:"last" binding:"required" validate:"gte=1,lte=1000000"`
	PrtstOnly    int    `body:"pretest" binding:"required" validate:"gte=0,lte=1"`
	ScorePrivate int    `body:"score_private" binding:"required" validate:"gte=0,lte=1"`
	Auth
}

func CtstEdit(ctx Context, param CtstEditParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	title := strings.TrimSpace(param.Title)
	start, err := time.Parse("2006-01-02 15:04:05", param.StartTime)
	if err != nil {
		ctx.JSONAPI(http.StatusBadRequest, "time format error", nil)
		return
	}
	if ctst.Finished {
		start = ctst.StartTime
		param.Duration = int(ctst.EndTime.Sub(ctst.StartTime).Minutes())
		param.PrtstOnly = libs.If(ctst.Pretest, 1, 0)
		param.ScorePrivate = libs.If(ctst.ScorePrivate, 1, 0)
	}
	err = internal.CTModify(param.CtstID, title, start, param.Duration, param.PrtstOnly, param.ScorePrivate)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPermGetParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstPermGet(ctx Context, param CtstPermGetParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	perms, err := internal.CTGetPermissions(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": perms})
	}
}

type CtstMgrGetParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstMgrGet(ctx Context, param CtstMgrGetParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	users, err := internal.CTGetManagers(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
	}
}

type CtstPermAddParam struct {
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
	PermID int `body:"permission_id" binding:"required" validate:"prmsid"`
	Auth
}

func CtstPermAdd(ctx Context, param CtstPermAddParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstMgrAddParam struct {
	CtstID    int `body:"contest_id" binding:"required" validate:"ctstid"`
	MgrUserID int `body:"user_id" binding:"required" validate:"userid"`
	Auth
}

func CtstMgrAdd(ctx Context, param CtstMgrAddParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	if !internal.USExists(param.MgrUserID) {
		ctx.JSONAPI(http.StatusBadRequest, "no such user id", nil)
		return
	}
	if internal.CTRegistered(param.CtstID, param.MgrUserID) {
		ctx.JSONAPI(http.StatusBadRequest, "user has registered this contest", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPermDelParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	PermID int `query:"permission_id" binding:"required" validate:"prmsid"`
	Auth
}

func CtstPermDel(ctx Context, param CtstPermDelParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstMgrDelParam struct {
	CtstID    int `query:"contest_id" binding:"required" validate:"ctstid"`
	MgrUserID int `query:"user_id" binding:"required" validate:"userid"`
	Auth
}

func CtstMgrDel(ctx Context, param CtstMgrDelParam) {
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPtcpGetParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstPtcpGet(ctx Context, param CtstPtcpGetParam) {
	if !param.CanSeeCtst(param.CtstID, param.CanEditCtst(param.CtstID)) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	parts, err := internal.CTGetParticipants(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": parts})
	}
}

type CtstSignupParam struct {
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
	Auth
}

func CtstSignup(ctx Context, param CtstSignupParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if !param.CanTakeCtst(contest, param.CanEditCtst(param.CtstID)) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	err = internal.CTAddParticipant(param.CtstID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstSignoutParam struct {
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	UserID int `session:"user_id"`
}

func CtstSignout(ctx Context, param CtstSignoutParam) {
	err := internal.CTDeleteParticipant(param.CtstID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstStandingParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstStanding(ctx Context, param CtstStandingParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	can_edit := param.CanEditCtst(param.CtstID)
	if !param.CanEnterCtst(ctst, param.CanEditCtst(param.CtstID)) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	raw_standing := internal.CTSGet(param.CtstID)
	standing := []internal.CTStandingEntry{}
	libs.DeepCopy(&standing, raw_standing)
	if !can_edit && ctst.EndTime.After(time.Now()) {
		if ctst.ScorePrivate {
			for _, v := range standing {
				if v.UserId == param.UserID {
					standing = []internal.CTStandingEntry{v}
					break
				}
			}
		}
		if ctst.Pretest {
			for k := range standing {
				standing[k].Scores = standing[k].SScores
				for i := range standing[k].Hacked {
					standing[k].Hacked[i] = false
				}
			}
		}
	}
	problems, err := internal.CTGetProblems(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"standing": standing, "problems": problems})
	}
}

type CtstFinishParam struct {
	Auth
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstFinish(ctx Context, param CtstFinishParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	if !param.CanEditCtst(param.CtstID) {
		ctx.JSONRPC(http.StatusForbidden, -32600, "", nil)
		return
	}
	if ctst.Finished {
		ctx.JSONRPC(http.StatusBadRequest, -32600, "Contest is already finished.", nil)
		return
	}
	if ctst.EndTime.After(time.Now()) {
		ctx.JSONRPC(http.StatusBadRequest, -32600, "Contest hasn't finished.", nil)
		return
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from submissions where contest_id=? and status>=0 and status<? limit 1", param.CtstID, internal.Finished)
	if count > 0 {
		ctx.JSONRPC(http.StatusBadRequest, -32600, "There are still some contest submissions judging, please wait.", nil)
		return
	}
	err = internal.CTFinish(param.CtstID)
	if err != nil {
		ctx.ErrorRPC(err)
	} else {
		ctx.JSONRPC(http.StatusOK, 0, "", nil)
	}
}

type CtstGetDashboardParam struct {
	ContestId int `query:"contest_id" validate:"required,ctstid"`
}

func CtstGetDashboard(ctx Context, param CtstGetDashboardParam) {
	ctx.JSONAPI(http.StatusOK, "", map[string]any{ "data": internal.CTDashboard(param.ContestId)})
}

type CtstAddDashboardParam struct {
	ContestId int    `body:"contest_id" validate:"required,ctstid"`
	Dashboard string `body:"dashboard" validate:"required,gte=1,lte=200"`
}

func CtstAddDashboard(ctx Context, param CtstAddDashboardParam) {
	err := internal.CTAddDashboard(param.ContestId, param.Dashboard)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}