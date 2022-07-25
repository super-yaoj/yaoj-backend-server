package services

import (
	"strings"
	"time"
	"yao/internal"
	"yao/libs"
)

func CTCanEdit(user_id int, user_group int, contest_id int) bool {
	if user_id < 0 {
		return false
	}
	if libs.IsAdmin(user_group) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -user_id)
	return count > 0
}

func CTCanSee(user_id int, contest_id int, can_edit bool) bool {
	if user_id < 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, libs.DefaultGroup)
		return count > 0
	}
	if can_edit {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select permission_id from contest_permissions where contest_id=?) as a join (select permission_id from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", contest_id, user_id)
	return count > 0
}

func CTCanTake(user_id int, contest internal.Contest, can_edit bool) bool {
	if user_id < 0 {
		return false
	}
	if contest.EndTime.After(time.Now()) {
		return !can_edit && CTCanSee(user_id, contest.Id, can_edit)
	}
	return false
}

func CTCanEnter(user_id int, contest internal.Contest, can_edit bool) bool {
	if can_edit {
		return true
	}
	if contest.StartTime.After(time.Now()) {
		return false
	} else if contest.EndTime.After(time.Now()) {
		return internal.CTRegistered(contest.Id, user_id)
	} else {
		return CTCanSee(user_id, contest.Id, can_edit)
	}
}

type CtstListParam struct {
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	UserID   int  `session:"user_id"`
	UserGrp  int  `session:"user_group"`
}

func CtstList(ctx Context, param CtstListParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	contests, isfull, err := internal.CTList(bound, param.PageSize, param.UserID, param.Left != nil, libs.IsAdmin(param.UserGrp))
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"isfull": isfull, "data": contests})
	}
}

type CtstGetParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstGet(ctx Context, param CtstGetParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	can_edit := CTCanEdit(param.UserID, param.UserGrp, param.CtstID)
	if !CTCanEnter(param.UserID, contest, can_edit) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	ctx.JSONAPI(200, "", map[string]any{"contest": contest, "can_edit": can_edit})
}

type CtstProbGetParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstProbGet(ctx Context, param CtstProbGetParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEnter(param.UserID, contest, CTCanEdit(param.UserID, param.UserGrp, param.CtstID)) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	problems, err := internal.CTGetProblems(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": problems})
	}
}

type CtstProbAddParam struct {
	CtstID  int `body:"contest_id" binding:"required"`
	ProbID  int `body:"problem_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstProbAdd(ctx Context, param CtstProbAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.PRExists(param.ProbID) {
		ctx.JSONAPI(400, "no such problem id", nil)
		return
	}
	err := internal.CTAddProblem(param.CtstID, param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstProbDelParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	ProbID  int `query:"problem_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstProbDel(ctx Context, param CtstProbDelParam) {
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
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
		ctx.JSONAPI(200, "", map[string]any{"id": id})
	}
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Greater_Than
// gte,lte 用在 string 上表示长度限制
type CtstEditParam struct {
	CtstID       int    `body:"contest_id" binding:"required"`
	Title        string `body:"title" validate:"gte=0,lte=190"`
	StartTime    string `body:"start_time"`
	Duration     int    `body:"last" binding:"required" validate:"gte=1,lte=1000000"`
	PrtstOnly    int    `body:"pretest" binding:"required" validate:"gte=0,lte=1"`
	ScorePrivate int    `body:"score_private" binding:"required" validate:"gte=0,lte=1"`
	UserID       int    `session:"user_id"`
	UserGrp      int    `session:"user_group"`
}

func CtstEdit(ctx Context, param CtstEditParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	title := strings.TrimSpace(param.Title)
	start, err := time.Parse("2006-01-02 15:04:05", param.StartTime)
	if err != nil {
		ctx.JSONAPI(400, "time format error", nil)
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
	CtstID  int `query:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstPermGet(ctx Context, param CtstPermGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	perms, err := internal.CTGetPermissions(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": perms})
	}
}

type CtstMgrGetParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstMgrGet(ctx Context, param CtstMgrGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	users, err := internal.CTGetManagers(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": users})
	}
}

type CtstPermAddParam struct {
	CtstID  int `body:"contest_id" binding:"required"`
	PermID  int `body:"permission_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstPermAdd(ctx Context, param CtstPermAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		ctx.JSONAPI(400, "no such permission id", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstMgrAddParam struct {
	CtstID    int `body:"contest_id" binding:"required"`
	UserID    int `body:"user_id" binding:"required"`
	CurUserID int `session:"user_id"`
	UserGrp   int `session:"user_group"`
}

func CtstMgrAdd(ctx Context, param CtstMgrAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.CurUserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.USExists(param.UserID) {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	}
	if internal.CTRegistered(param.CtstID, param.UserID) {
		ctx.JSONAPI(400, "user has registered this contest", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, -param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPermDelParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	PermID  int `query:"permission_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstPermDel(ctx Context, param CtstPermDelParam) {
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstMgrDelParam struct {
	CtstID    int `query:"contest_id" binding:"required"`
	UserID    int `query:"user_id" binding:"required"`
	CurUserID int `session:"user_id"`
	UserGrp   int `session:"user_group"`
}

func CtstMgrDel(ctx Context, param CtstMgrDelParam) {
	if !CTCanEdit(param.CurUserID, param.UserGrp, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, -param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPtcpGetParam struct {
	CtstID  int `query:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstPtcpGet(ctx Context, param CtstPtcpGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanSee(param.UserID, param.CtstID, CTCanEdit(param.UserID, param.UserGrp, param.CtstID)) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	parts, err := internal.CTGetParticipants(param.CtstID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": parts})
	}
}

type CtstSignupParam struct {
	CtstID  int `body:"contest_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstSignup(ctx Context, param CtstSignupParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanTake(param.UserID, contest, CTCanEdit(param.UserID, param.UserGrp, param.CtstID)) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err = internal.CTAddParticipant(param.CtstID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstSignoutParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	UserID int `session:"user_id"`
}

func CtstSignout(ctx Context, param CtstSignoutParam) {
	err := internal.CTDeleteParticipant(param.CtstID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstStandingParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	UserID int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstStanding(ctx Context, param CtstStandingParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	can_edit := CTCanEdit(param.UserID, param.UserGrp, param.CtstID)
	if !CTCanEnter(param.UserID, ctst, can_edit) {
		ctx.JSONAPI(403, "", nil)
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
		ctx.JSONAPI(200, "", map[string]any{"standing": standing, "problems": problems})
	}
}

type CtstFinishParam struct {
	CtstID int `body:"contest_id" binding:"required"`
	UserID int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func CtstFinish(ctx Context, param CtstFinishParam) {
	ctst, err := internal.CTQuery(param.CtstID, -1)
	if err != nil {
		ctx.JSONRPC(404, -32600, "", nil)
		return
	}
	if !CTCanEdit(param.UserID, param.UserGrp, param.CtstID) {
		ctx.JSONRPC(403, -32600, "", nil)
		return
	}
	if ctst.Finished {
		ctx.JSONRPC(400, -32600, "Contest is already finished.", nil)
		return
	}
	if ctst.EndTime.After(time.Now()) {
		ctx.JSONRPC(400, -32600, "Contest hasn't finished.", nil)
		return
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from submissions where contest_id=? and status>=0 and status<? limit 1", param.CtstID, internal.Finished)
	if count > 0 {
		ctx.JSONRPC(400, -32600, "There are still some contest submissions judging, please wait.", nil)
		return
	}
	err = internal.CTFinish(param.CtstID)
	if err != nil {
		ctx.ErrorRPC(err)
	} else {
		ctx.JSONRPC(200, 0, "", nil)
	}
}