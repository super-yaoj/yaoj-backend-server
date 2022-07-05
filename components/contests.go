package components

import (
	"strings"
	"time"
	"yao/controllers"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func CTCanEdit(ctx *gin.Context, contest_id int) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 { return false }
	if ISAdmin(ctx) { return true }
	count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -user_id)
	return count > 0
}

func CTCanSee(ctx *gin.Context, contest_id int, can_edit bool) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, libs.DefaultGroup)
		return count > 0
	}
	if can_edit { return true }
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select permission_id from contest_permissions where contest_id=?) as a join (select permission_id from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", contest_id, user_id)
	return count > 0
}

func CTCanTake(ctx *gin.Context, contest controllers.Contest, can_edit bool) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 { return false }
	if contest.EndTime.After(time.Now()) {
		return !can_edit && CTCanSee(ctx, contest.Id, can_edit)
	}
	return false
}

func CTCanEnter(ctx *gin.Context, contest controllers.Contest, can_edit bool) bool {
	user_id := GetUserId(ctx);
	if can_edit { return true }
	if contest.StartTime.After(time.Now()) {
		return false;
	} else if contest.EndTime.After(time.Now()) {
		return controllers.CTRegistered(contest.Id, user_id)
	} else {
		return CTCanSee(ctx, contest.Id, can_edit)
	}
}

func CTList(ctx *gin.Context) {
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok { return }
	user_id := GetUserId(ctx)
	_, isleft := ctx.GetQuery("left")
	bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	if !ok { return }
	contests, isfull, err := controllers.CTList(bound, pagesize, user_id, isleft, ISAdmin(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "isfull": isfull, "data": contests })
	}
}

func CTQuery(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	contest, err := controllers.CTQuery(contest_id, GetUserId(ctx))
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	can_edit := CTCanEdit(ctx, contest_id)
	if !CTCanEnter(ctx, contest, can_edit) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{ "contest": contest, "can_edit": can_edit })
}

func CTGetProblems(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	contest, err := controllers.CTQuery(contest_id, GetUserId(ctx))
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEnter(ctx, contest, CTCanEdit(ctx, contest_id)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	problems, err := controllers.CTGetProblems(contest_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": problems })
	}
}

func CTAddProblem(ctx *gin.Context) {
	contest_id, ok := libs.PostInt(ctx, "contest_id")
	if !ok { return }
	problem_id, ok := libs.PostInt(ctx, "problem_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !controllers.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 400, "no such problem id", nil)
		return
	}
	err := controllers.CTAddProblem(contest_id, problem_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTDeleteProblem(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	problem_id, ok := libs.GetInt(ctx, "problem_id")
	if !ok { return }
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := controllers.CTDeleteProblem(contest_id, problem_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTCreate(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, err := controllers.CTCreate()
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "id": id })
	}
}

func CTModify(ctx *gin.Context) {
	contest_id, ok := libs.PostInt(ctx, "contest_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	title := strings.TrimSpace(ctx.PostForm("title"))
	if len(title) == 0 || len(title) > 190 {
		libs.APIWriteBack(ctx, 400, "title too long", nil)
		return
	}
	start, err := time.Parse("2006-01-02 15:04:05", ctx.PostForm("start_time"))
	if err != nil {
		libs.APIWriteBack(ctx, 400, "time format error", nil)
		return
	}
	last, ok := libs.PostIntRange(ctx, "last", 1, 1000000)
	if !ok { return }
	pretest, ok := libs.PostIntRange(ctx, "pretest", 0, 1)
	if !ok { return }
	score_private, ok := libs.PostIntRange(ctx, "score_private", 0, 1)
	if !ok { return }
	err = controllers.CTModify(contest_id, title, start, last, pretest, score_private)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTGetPermissions(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	perms, err := controllers.CTGetPermissions(contest_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": perms })
	}
}

func CTGetManagers(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	users, err := controllers.CTGetManagers(contest_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": users })
	}
}

func CTAddPermission(ctx *gin.Context) {
	contest_id, ok := libs.PostInt(ctx, "contest_id")
	if !ok { return }
	permission_id, ok := libs.PostInt(ctx, "permission_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !controllers.PMExists(permission_id) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := controllers.CTAddPermission(contest_id, permission_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTAddManager(ctx *gin.Context) {
	contest_id, ok := libs.PostInt(ctx, "contest_id")
	if !ok { return }
	user_id, ok := libs.PostInt(ctx, "user_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !controllers.USExists(user_id) {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	}
	if controllers.CTRegistered(contest_id, user_id) {
		libs.APIWriteBack(ctx, 400, "user has registered this contest", nil)
		return
	}
	err := controllers.CTAddPermission(contest_id, -user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTDeletePermission(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	permission_id, ok := libs.GetInt(ctx, "permission_id")
	if !ok { return }
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := controllers.CTDeletePermission(contest_id, permission_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTDeleteManager(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	user_id, ok := libs.GetInt(ctx, "user_id")
	if !ok { return }
	if !CTCanEdit(ctx, contest_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := controllers.CTDeletePermission(contest_id, -user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTGetParticipants(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	if !controllers.CTExists(contest_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanSee(ctx, contest_id, CTCanEdit(ctx, contest_id)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	parts, err := controllers.CTGetParticipants(contest_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": parts })
	}
}

func CTSignup(ctx *gin.Context) {
	contest_id, ok := libs.PostInt(ctx, "contest_id")
	if !ok { return }
	user_id := GetUserId(ctx)
	contest, err := controllers.CTQuery(contest_id, user_id)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !CTCanTake(ctx, contest, CTCanEdit(ctx, contest_id)) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err = controllers.CTAddParticipant(contest_id, user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func CTSignout(ctx *gin.Context) {
	contest_id, ok := libs.GetInt(ctx, "contest_id")
	if !ok { return }
	user_id := GetUserId(ctx)
	if !ok { return }
	err := controllers.CTDeleteParticipant(contest_id, user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}