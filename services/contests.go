package services

import (
	"net/http"
	"strings"
	"time"
	"yao/db"
	"yao/internal"

	utils "github.com/super-yaoj/yaoj-utils"
)

type CtstListParam struct {
	Auth
	Page `validate:"pagecanbound"`
}

func CtstList(ctx Context, param CtstListParam) {
	contests, isfull, err := internal.CTList(
		param.Page.Bound(), *param.PageSize, param.UserID, param.IsLeft(), internal.IsAdmin(param.UserGrp),
	)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"isfull": isfull, "data": contests})
	}
}

type CtstGetParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstGet(ctx Context, param CtstGetParam) {
	param.NewPermit().TryEnterCtst(param.CtstID).Success(func(a any) {
		ctst := a.(PermitCtst)
		ctx.JSONAPI(http.StatusOK, "", map[string]any{
			"contest":  ctst.Contest,
			"can_edit": ctst.CanEdit,
		})
	}).FailAPIStatusForbidden(ctx)
}

type CtstProbGetParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstProbGet(ctx Context, param CtstProbGetParam) {
	param.NewPermit().TryEnterCtst(param.CtstID).Success(func(any) {
		problems, err := internal.CTGetProblems(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": problems})
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstProbAddParam struct {
	Auth
	CtstID int `body:"contest_id" validate:"required,ctstid"`
	ProbID int `body:"problem_id" validate:"required,probid"`
}

func CtstProbAdd(ctx Context, param CtstProbAddParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		err := internal.CTAddProblem(param.CtstID, param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstProbDelParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
	ProbID int `query:"problem_id" validate:"required,probid"`
}

func CtstProbDel(ctx Context, param CtstProbDelParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		err := internal.CTDeleteProblem(param.CtstID, param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstCreateParam struct {
	Auth
}

func CtstCreate(ctx Context, param CtstCreateParam) {
	param.NewPermit().AsAdmin().Success(func(any) {
		id, err := internal.CTCreate()
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
		}
	}).FailAPIStatusForbidden(ctx)
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Greater_Than
// gte,lte 用在 string 上表示长度限制
type CtstEditParam struct {
	Auth
	CtstID       int    `body:"contest_id" validate:"required,ctstid"`
	Title        string `body:"title" validate:"required,gte=1,lte=190"`
	StartTime    string `body:"start_time" validate:"required"`
	Duration     int    `body:"last" validate:"required,gte=1,lte=1000000"`
	PrtstOnly    int    `body:"pretest" validate:"gte=0,lte=1"`
	ScorePrivate int    `body:"score_private" validate:"gte=0,lte=1"`
}

func CtstEdit(ctx Context, param CtstEditParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
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
			param.PrtstOnly = utils.If(ctst.Pretest, 1, 0)
			param.ScorePrivate = utils.If(ctst.ScorePrivate, 1, 0)
		}
		err = internal.CTModify(param.CtstID, title, start, param.Duration, param.PrtstOnly, param.ScorePrivate)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstPermGetParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstPermGet(ctx Context, param CtstPermGetParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		perms, err := internal.CTGetPermissions(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": perms})
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstMgrGetParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstMgrGet(ctx Context, param CtstMgrGetParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		users, err := internal.CTGetManagers(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstPermAddParam struct {
	Auth
	CtstID int `body:"contest_id" validate:"required,ctstid"`
	PermID int `body:"permission_id" validate:"required,prmsid"`
}

func CtstPermAdd(ctx Context, param CtstPermAddParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		err := internal.CTAddPermission(param.CtstID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstMgrAddParam struct {
	Auth
	CtstID    int `body:"contest_id" validate:"required,ctstid"`
	MgrUserID int `body:"user_id" validate:"required,userid"`
}

func CtstMgrAdd(ctx Context, param CtstMgrAddParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		if !internal.UserExists(param.MgrUserID) {
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
	}).FailAPIStatusForbidden(ctx)
}

type CtstPermDelParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
	PermID int `query:"permission_id" validate:"required,prmsid"`
}

func CtstPermDel(ctx Context, param CtstPermDelParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		err := internal.CTDeletePermission(param.CtstID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstMgrDelParam struct {
	Auth
	CtstID    int `query:"contest_id" validate:"required,ctstid"`
	MgrUserID int `query:"user_id" validate:"required,userid"`
}

func CtstMgrDel(ctx Context, param CtstMgrDelParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		err := internal.CTDeletePermission(param.CtstID, -param.MgrUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstPtcpGetParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstPtcpGet(ctx Context, param CtstPtcpGetParam) {
	param.NewPermit().TrySeeCtst(param.CtstID).Success(func(any) {
		parts, err := internal.CTGetParticipants(param.CtstID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": parts})
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstSignupParam struct {
	Auth
	CtstID int `body:"contest_id" validate:"required,ctstid"`
}

func CtstSignup(ctx Context, param CtstSignupParam) {
	param.NewPermit().TryTakeCtst(param.CtstID).Success(func(any) {
		err := internal.CTAddParticipant(param.CtstID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type CtstSignoutParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstSignout(ctx Context, param CtstSignoutParam) {
	err := internal.CTDeleteParticipant(param.CtstID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstStandingParam struct {
	Auth
	CtstID int `query:"contest_id" validate:"required,ctstid"`
}

func CtstStanding(ctx Context, param CtstStandingParam) {
	param.NewPermit().TryEnterCtst(param.CtstID).Success(func(a any) {
		ctst := a.(PermitCtst)
		raw_standing := internal.CTSGet(param.CtstID)
		standing := []internal.CTStandingEntry{}
		utils.DeepCopy(&standing, raw_standing)
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
	}).FailAPIStatusForbidden(ctx)
}

type CtstFinishParam struct {
	Auth
	CtstID int `body:"contest_id" validate:"required,ctstid"`
}

func CtstFinish(ctx Context, param CtstFinishParam) {
	param.NewPermit().TryEditCtst(param.CtstID).Success(func(any) {
		ctst, err := internal.CTQuery(param.CtstID, -1)
		if ctst.Finished {
			ctx.JSONRPC(http.StatusBadRequest, -32600, "Contest is already finished.", nil)
			return
		}
		if ctst.EndTime.After(time.Now()) {
			ctx.JSONRPC(http.StatusBadRequest, -32600, "Contest hasn't finished.", nil)
			return
		}
		count, _ := db.SelectSingleInt("select count(*) from submissions where contest_id=? and status>=0 and status<? limit 1", param.CtstID, internal.Finished)
		if count > 0 {
			ctx.JSONRPC(http.StatusBadRequest, -32600, "There are still some contest submissions judging, please wait.", nil)
			return
		}
		err = internal.CTFinish(param.CtstID)
		if err != nil {
			ctx.ErrorRPC(err)
		}
	}).FailRPCStatusForbidden(ctx)
}

type CtstGetDashboardParam struct {
	Auth
	ContestId int `query:"contest_id" validate:"required,ctstid"`
}

func CtstGetDashboard(ctx Context, param CtstGetDashboardParam) {
	param.NewPermit().TryEnterCtst(param.ContestId).Success(func(any) {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": internal.CTDashboard(param.ContestId)})
	}).FailAPIStatusForbidden(ctx)
}

type CtstAddDashboardParam struct {
	Auth
	ContestId int    `body:"contest_id" validate:"required,ctstid"`
	Dashboard string `body:"dashboard" validate:"required,gte=1,lte=200"`
}

func CtstAddDashboard(ctx Context, param CtstAddDashboardParam) {
	param.NewPermit().TryEditCtst(param.ContestId).Success(func(any) {
		err := internal.CTAddDashboard(param.ContestId, param.Dashboard)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}
