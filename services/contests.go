package services

import (
	"strings"
	"time"
	"yao/internal"
	"yao/libs"
)

func CTCanEdit(auth Auth, contest_id int) bool {
	if auth.UserID < 0 {
		return false
	}
	if libs.IsAdmin(auth.UserGrp) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -auth.UserID)
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
		ctx.JSONAPI(200, "", map[string]any{"isfull": isfull, "data": contests})
	}
}

type CtstGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	Auth
}

func CtstGet(ctx Context, param CtstGetParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	can_edit := CTCanEdit(param.Auth, param.CtstID)
	if !CTCanEnter(param.UserID, contest, can_edit) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	ctx.JSONAPI(200, "", map[string]any{"contest": contest, "can_edit": can_edit})
}

type CtstProbGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	Auth
}

func CtstProbGet(ctx Context, param CtstProbGetParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEnter(param.UserID, contest, CTCanEdit(param.Auth, param.CtstID)) {
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
	CtstID int `body:"contest_id" binding:"required"`
	ProbID int `body:"problem_id" binding:"required" validate:"probid"`
	Auth
}

func CtstProbAdd(ctx Context, param CtstProbAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.CTAddProblem(param.CtstID, param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstProbDelParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	ProbID int `query:"problem_id" binding:"required"`
	Auth
}

func CtstProbDel(ctx Context, param CtstProbDelParam) {
	if !CTCanEdit(param.Auth, param.CtstID) {
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
	Auth
}

func CtstEdit(ctx Context, param CtstEditParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	title := strings.TrimSpace(param.Title)
	start, err := time.Parse("2006-01-02 15:04:05", param.StartTime)
	if err != nil {
		ctx.JSONAPI(400, "time format error", nil)
		return
	}
	err = internal.CTModify(param.CtstID, title, start, param.Duration, param.PrtstOnly, param.ScorePrivate)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPermGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	Auth
}

func CtstPermGet(ctx Context, param CtstPermGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
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
	CtstID int `query:"contest_id" binding:"required"`
	Auth
}

func CtstMgrGet(ctx Context, param CtstMgrGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
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
	CtstID int `body:"contest_id" binding:"required"`
	PermID int `body:"permission_id" binding:"required"`
	Auth
}

func CtstPermAdd(ctx Context, param CtstPermAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
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
	MgrUserID int `body:"user_id" binding:"required"`
	Auth
}

func CtstMgrAdd(ctx Context, param CtstMgrAddParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanEdit(param.Auth, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.USExists(param.MgrUserID) {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	}
	if internal.CTRegistered(param.CtstID, param.MgrUserID) {
		ctx.JSONAPI(400, "user has registered this contest", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPermDelParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	PermID int `query:"permission_id" binding:"required"`
	Auth
}

func CtstPermDel(ctx Context, param CtstPermDelParam) {
	if !CTCanEdit(param.Auth, param.CtstID) {
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
	MgrUserID int `query:"user_id" binding:"required"`
	Auth
}

func CtstMgrDel(ctx Context, param CtstMgrDelParam) {
	if !CTCanEdit(param.Auth, param.CtstID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type CtstPtcpGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	Auth
}

func CtstPtcpGet(ctx Context, param CtstPtcpGetParam) {
	if !internal.CTExists(param.CtstID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanSee(param.UserID, param.CtstID, CTCanEdit(param.Auth, param.CtstID)) {
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
	CtstID int `body:"contest_id" binding:"required"`
	Auth
}

func CtstSignup(ctx Context, param CtstSignupParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !CTCanTake(param.UserID, contest, CTCanEdit(param.Auth, param.CtstID)) {
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
