package services

import (
	"strings"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func CTCanEdit(ctx *gin.Context, contest_id int) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		return false
	}
	if ISAdmin(ctx) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -user_id)
	return count > 0
}

func CTCanSee(ctx *gin.Context, contest_id int, can_edit bool) bool {
	user_id := GetUserId(ctx)
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

func CTCanTake(ctx *gin.Context, contest internal.Contest, can_edit bool) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		return false
	}
	if contest.EndTime.After(time.Now()) {
		return !can_edit && CTCanSee(ctx, contest.Id, can_edit)
	}
	return false
}

func CTCanEnter(ctx *gin.Context, contest internal.Contest, can_edit bool) bool {
	user_id := GetUserId(ctx)
	if can_edit {
		return true
	}
	if contest.StartTime.After(time.Now()) {
		return false
	} else if contest.EndTime.After(time.Now()) {
		return internal.CTRegistered(contest.Id, user_id)
	} else {
		return CTCanSee(ctx, contest.Id, can_edit)
	}
}

type CtstListParam struct {
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	UserID   int  `session:"user_id"`
}

func CtstList(ctx *gin.Context, param CtstListParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	contests, isfull, err := internal.CTList(bound, param.PageSize, param.UserID, param.Left != nil, ISAdmin(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"isfull": isfull, "data": contests})
	}
}

type CtstGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
}

func CtstGet(ctx *gin.Context, param CtstGetParam) {
	contest, err := internal.CTQuery(param.CtstID, GetUserId(ctx))
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	can_edit := CTCanEdit(ctx, param.CtstID)
	if !CTCanEnter(ctx, contest, can_edit) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"contest": contest, "can_edit": can_edit})
}

type CtstProbGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
}

func CtstProbGet(ctx *gin.Context, param CtstProbGetParam) {
	contest, err := internal.CTQuery(param.CtstID, GetUserId(ctx))
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEnter(ctx, contest, CTCanEdit(ctx, param.CtstID)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	problems, err := internal.CTGetProblems(param.CtstID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": problems})
	}
}

type CtstProbAddParam struct {
	CtstID int `body:"contest_id" binding:"required"`
	ProbID int `body:"problem_id" binding:"required"`
}

func CtstProbAdd(ctx *gin.Context, param CtstProbAddParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 400, "no such problem id", nil)
		return
	}
	err := internal.CTAddProblem(param.CtstID, param.ProbID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstProbDelParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	ProbID int `query:"problem_id" binding:"required"`
}

func CtstProbDel(ctx *gin.Context, param CtstProbDelParam) {
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.CTDeleteProblem(param.CtstID, param.ProbID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTCreate(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, err := internal.CTCreate()
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
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
}

func CtstEdit(ctx *gin.Context, param CtstEditParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	title := strings.TrimSpace(param.Title)
	start, err := time.Parse("2006-01-02 15:04:05", param.StartTime)
	if err != nil {
		libs.APIWriteBack(ctx, 400, "time format error", nil)
		return
	}
	err = internal.CTModify(param.CtstID, title, start, param.Duration, param.PrtstOnly, param.ScorePrivate)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstPermGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
}

func CtstPermGet(ctx *gin.Context, param CtstPermGetParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	perms, err := internal.CTGetPermissions(param.CtstID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": perms})
	}
}

type CtstMgrGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
}

func CtstMgrGet(ctx *gin.Context, param CtstMgrGetParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	users, err := internal.CTGetManagers(param.CtstID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users})
	}
}

type CtstPermAddParam struct {
	CtstID int `body:"contest_id" binding:"required"`
	PermID int `body:"permission_id" binding:"required"`
}

func CtstPermAdd(ctx *gin.Context, param CtstPermAddParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstMgrAddParam struct {
	CtstID int `body:"contest_id" binding:"required"`
	UserID int `body:"user_id" binding:"required"`
}

func CtstMgrAdd(ctx *gin.Context, param CtstMgrAddParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.USExists(param.UserID) {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	}
	if internal.CTRegistered(param.CtstID, param.UserID) {
		libs.APIWriteBack(ctx, 400, "user has registered this contest", nil)
		return
	}
	err := internal.CTAddPermission(param.CtstID, -param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstPermDelParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	PermID int `query:"permission_id" binding:"required"`
}

func CtstPermDel(ctx *gin.Context, param CtstPermDelParam) {
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstMgrDelParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	UserID int `query:"user_id" binding:"required"`
}

func CtstMgrDel(ctx *gin.Context, param CtstMgrDelParam) {
	if !CTCanEdit(ctx, param.CtstID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.CTDeletePermission(param.CtstID, -param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstPtcpGetParam struct {
	CtstID int `query:"contest_id" binding:"required"`
}

func CtstPtcpGet(ctx *gin.Context, param CtstPtcpGetParam) {
	if !internal.CTExists(param.CtstID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanSee(ctx, param.CtstID, CTCanEdit(ctx, param.CtstID)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	parts, err := internal.CTGetParticipants(param.CtstID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": parts})
	}
}

type CtstSignupParam struct {
	CtstID int `body:"contest_id" binding:"required"`
	UserID int `session:"user_id"`
}

func CtstSignup(ctx *gin.Context, param CtstSignupParam) {
	contest, err := internal.CTQuery(param.CtstID, param.UserID)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanTake(ctx, contest, CTCanEdit(ctx, param.CtstID)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err = internal.CTAddParticipant(param.CtstID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type CtstSignoutParam struct {
	CtstID int `query:"contest_id" binding:"required"`
	UserID int `session:"user_id"`
}

func CtstSignout(ctx *gin.Context, param CtstSignoutParam) {
	err := internal.CTDeleteParticipant(param.CtstID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}
