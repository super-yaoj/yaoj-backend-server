package services

import (
	"net/http"
	"strings"
	"time"
	"yao/internal"
	"yao/libs"
)

type CtstListParam struct {
	Page `validate:"pagecanbound"`
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
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstGet(ctx Context, param CtstGetParam) {
	param.TryEnterCtst(param.CtstID).Then(func(ctst PermitCtst) {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{
			"contest":  ctst.Contest,
			"can_edit": ctst.CanEdit,
		})
	}).ElseAPIStatusForbidden(ctx)
}

type CtstProbGetParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstProbGet(ctx Context, param CtstProbGetParam) {
	param.TryEnterCtst(param.CtstID).Then(func(ctst PermitCtst) {
		problems, err := internal.CTGetProblems(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": problems})
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstProbAddParam struct {
	Auth
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
	ProbID int `body:"problem_id" binding:"required" validate:"probid"`
}

func CtstProbAdd(ctx Context, param CtstProbAddParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		err := internal.CTAddProblem(param.CtstID, param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstProbDelParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
}

func CtstProbDel(ctx Context, param CtstProbDelParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		err := internal.CTDeleteProblem(param.CtstID, param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstCreateParam struct {
	Auth
}

func CtstCreate(ctx Context, param CtstCreateParam) {
	param.AsAdmin().Then(func(struct{}) {
		id, err := internal.CTCreate()
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
		}
	}).ElseAPIStatusForbidden(ctx)
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Greater_Than
// gte,lte 用在 string 上表示长度限制
type CtstEditParam struct {
	Auth
	CtstID       int    `body:"contest_id" binding:"required" validate:"ctstid"`
	Title        string `body:"title" validate:"gte=0,lte=190"`
	StartTime    string `body:"start_time"`
	Duration     int    `body:"last" binding:"required" validate:"gte=1,lte=1000000"`
	PrtstOnly    int    `body:"pretest" binding:"required" validate:"gte=0,lte=1"`
	ScorePrivate int    `body:"score_private" binding:"required" validate:"gte=0,lte=1"`
}

func CtstEdit(ctx Context, param CtstEditParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		ctst, err := internal.CTQuery(param.CtstID, -1)
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
	}).ElseAPIStatusForbidden(ctx)
}

type CtstPermGetParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstPermGet(ctx Context, param CtstPermGetParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		perms, err := internal.CTGetPermissions(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": perms})
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstMgrGetParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstMgrGet(ctx Context, param CtstMgrGetParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		users, err := internal.CTGetManagers(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstPermAddParam struct {
	Auth
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
	PermID int `body:"permission_id" binding:"required" validate:"prmsid"`
}

func CtstPermAdd(ctx Context, param CtstPermAddParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		err := internal.CTAddPermission(param.CtstID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstMgrAddParam struct {
	Auth
	CtstID    int `body:"contest_id" binding:"required" validate:"ctstid"`
	MgrUserID int `body:"user_id" binding:"required" validate:"userid"`
}

func CtstMgrAdd(ctx Context, param CtstMgrAddParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
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
	}).ElseAPIStatusForbidden(ctx)
}

type CtstPermDelParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
	PermID int `query:"permission_id" binding:"required" validate:"prmsid"`
}

func CtstPermDel(ctx Context, param CtstPermDelParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		err := internal.CTDeletePermission(param.CtstID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstMgrDelParam struct {
	Auth
	CtstID    int `query:"contest_id" binding:"required" validate:"ctstid"`
	MgrUserID int `query:"user_id" binding:"required" validate:"userid"`
}

func CtstMgrDel(ctx Context, param CtstMgrDelParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		err := internal.CTDeletePermission(param.CtstID, -param.MgrUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstPtcpGetParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstPtcpGet(ctx Context, param CtstPtcpGetParam) {
	param.TrySeeCtst(param.CtstID).Then(func(bool) {
		parts, err := internal.CTGetParticipants(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": parts})
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstSignupParam struct {
	Auth
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstSignup(ctx Context, param CtstSignupParam) {
	param.TryTakeCtst(param.CtstID).Then(func(ctst PermitCtst) {
		err := internal.CTAddParticipant(param.CtstID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}

type CtstSignoutParam struct {
	Auth
	CtstID int `query:"contest_id" binding:"required" validate:"ctstid"`
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
	param.TryEnterCtst(param.CtstID).Then(func(ctst PermitCtst) {
		raw_standing := internal.CTSGet(param.CtstID)
		standing := []internal.CTStandingEntry{}
		libs.DeepCopy(&standing, raw_standing)
		if !ctst.CanEdit && ctst.EndTime.After(time.Now()) {
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
	}).ElseAPIStatusForbidden(ctx)
}

type CtstFinishParam struct {
	Auth
	CtstID int `body:"contest_id" binding:"required" validate:"ctstid"`
}

func CtstFinish(ctx Context, param CtstFinishParam) {
	param.TryEditCtst(param.CtstID).Then(func(struct{}) {
		ctst, err := internal.CTQuery(param.CtstID, -1)
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
		}
	}).ElseRPCStatusForbidden(ctx)
}

type CtstGetDashboardParam struct {
	Auth
	ContestId int `query:"contest_id" validate:"required,ctstid"`
}

func CtstGetDashboard(ctx Context, param CtstGetDashboardParam) {
	param.TryEnterCtst(param.ContestId).Then(func(a PermitCtst) {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": internal.CTDashboard(param.ContestId)})
	}).ElseAPIStatusForbidden(ctx)
}

type CtstAddDashboardParam struct {
	Auth
	ContestId int    `body:"contest_id" validate:"required,ctstid"`
	Dashboard string `body:"dashboard" validate:"required,gte=1,lte=200"`
}

func CtstAddDashboard(ctx Context, param CtstAddDashboardParam) {
	param.TryEditCtst(param.ContestId).Then(func(struct{}) {
		err := internal.CTAddDashboard(param.ContestId, param.Dashboard)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).ElseAPIStatusForbidden(ctx)
}
