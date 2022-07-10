package services

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
	"github.com/k0kubun/pp"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
)

func PRCanEdit(ctx *gin.Context, problem_id int) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		return false
	}
	if ISAdmin(ctx) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, -user_id)
	return count > 0
}

func PRCanSeeWithoutContest(ctx *gin.Context, problem_id int) bool {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, libs.DefaultGroup)
		return count > 0
	}
	if PRCanEdit(ctx, problem_id) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select * from problem_permissions where problem_id=?) as a join (select * from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", problem_id, user_id)
	return count > 0
}

/*
 */
func PRCanSeeFromContest(ctx *gin.Context, problem_id, contest_id int) bool {
	contest, _ := internal.CTQuery(contest_id, GetUserId(ctx))
	if CTCanEnter(ctx, contest, CTCanEdit(ctx, contest_id)) &&
		contest.StartTime.Before(time.Now()) && contest.EndTime.After(time.Now()) {
		return internal.CTHasProblem(contest_id, problem_id)
	}
	return false
}

/*
args: contest_id=0 means not in contest

return: (must see from contest, can see)
*/
func PRCanSee(ctx *gin.Context, problem_id, contest_id int) (bool, bool) {
	if !PRCanSeeWithoutContest(ctx, problem_id) {
		if contest_id <= 0 || !PRCanSeeFromContest(ctx, problem_id, contest_id) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return false, false
		}
		return true, true
	}
	return false, true
}

func PRList(ctx *gin.Context) {
	user_id := GetUserId(ctx)
	_, isleft := ctx.GetQuery("left")
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok {
		return
	}
	bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	if !ok {
		return
	}
	problems, isfull, err := internal.PRList(bound, pagesize, user_id, isleft, ISAdmin(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"isfull": isfull, "data": problems})
	}
}

func PRCreate(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, err := libs.DBInsertGetId("insert into problems values (null, \"New Problem\", 0, \"\", \"\")")
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
	}
}

// 查询问题
type ProbGetParam struct {
	ProbID  *int `query:"problem_id"`
	CtstID  int  `query:"contest_id"`
	UserID  int  `session:"user_id"`
	UserGrp int  `session:"user_group"`
}

func ProbGet(ctx *gin.Context, param ProbGetParam) {
	if param.ProbID == nil {
		return
	}
	if !internal.PRExists(*param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	in_contest, ok := PRCanSee(ctx, *param.ProbID, param.CtstID)
	if !ok {
		return
	}
	prob, err := internal.PRQuery(*param.ProbID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	if in_contest {
		prob.Tutorial_zh = ""
		prob.Tutorial_en = ""
	}
	can_edit := PRCanEdit(ctx, *param.ProbID)
	if !can_edit {
		prob.DataInfo = problem.DataInfo{}
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"problem": *prob, "can_edit": can_edit})
}

// 获取题目权限
type ProbGetPermParam struct {
	ProbID *int `query:"problem_id"`
}

func ProbGetPerm(ctx *gin.Context, param ProbGetPermParam) {
	if param.ProbID == nil {
		return
	}
	if !internal.PRExists(*param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, *param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	pers, err := internal.PRGetPermissions(*param.ProbID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": pers})
	}
}

type ProbAddPermParam struct {
	ProbID *int `body:"problem_id"`
	PermID *int `body:"permission_id"`
}

func ProbAddPerm(ctx *gin.Context, param ProbAddPermParam) {
	pp.Print(param)
	if param.ProbID == nil {
		return
	}
	if param.PermID == nil {
		return
	}
	if !internal.PRExists(*param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, *param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.PMExists(*param.PermID) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := internal.PRAddPermission(*param.ProbID, *param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type ProbDelPermParam struct {
	ProbID *int `query:"problem_id"`
	PermID *int `query:"problem_id"`
}

func ProbDelPerm(ctx *gin.Context, param ProbDelPermParam) {
	if param.ProbID == nil {
		return
	}
	if param.PermID == nil {
		return
	}
	if !PRCanEdit(ctx, *param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.PRDeletePermission(*param.ProbID, *param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func PRGetManagers(ctx *gin.Context) {
	problem_id, ok := libs.GetInt(ctx, "problem_id")
	if !ok {
		return
	}
	if !internal.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, problem_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	users, err := internal.PRGetManagers(problem_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users})
	}
}

func PRAddManager(ctx *gin.Context) {
	problem_id, ok := libs.PostInt(ctx, "problem_id")
	if !ok {
		return
	}
	user_id, ok := libs.PostInt(ctx, "user_id")
	if !ok {
		return
	}
	if !internal.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, problem_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.USExists(user_id) {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	}
	err := internal.PRAddPermission(problem_id, -user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func PRDeleteManager(ctx *gin.Context) {
	problem_id, ok := libs.GetInt(ctx, "problem_id")
	if !ok {
		return
	}
	user_id, ok := libs.GetInt(ctx, "user_id")
	if !ok {
		return
	}
	if !PRCanEdit(ctx, problem_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.PRDeletePermission(problem_id, -user_id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func PRPutData(ctx *gin.Context) {
	problem_id, ok := libs.PostInt(ctx, "problem_id")
	if !ok {
		return
	}
	if !internal.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, problem_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	file, err := ctx.FormFile("data")
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
		return
	}
	ext := strings.Split(file.Filename, ".")
	if ext[len(ext)-1] != "zip" {
		libs.APIWriteBack(ctx, 400, "doesn't support file extension "+ext[len(ext)-1], nil)
		return
	}

	tmpdir := libs.GetTempDir()
	defer os.RemoveAll(tmpdir)
	err = ctx.SaveUploadedFile(file, path.Join(tmpdir, "1.zip"))
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	log.Printf("file uploaded saves at %s", path.Join(tmpdir, "1.zip"))
	err = internal.PRPutData(problem_id, tmpdir)
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
	}
}

func PRDownloadData(ctx *gin.Context) {
	problem_id, ok := libs.GetInt(ctx, "problem_id")
	if !ok {
		return
	}
	if !internal.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	t := ctx.Query("type")
	internal.ProblemRWLock.RLock(problem_id)
	defer internal.ProblemRWLock.RUnlock(problem_id)
	if t == "data" {
		if !PRCanEdit(ctx, problem_id) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		path := internal.PRGetDataZip(problem_id)
		_, err := os.Stat(path)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("problem_%d.zip", problem_id))
		}
	} else {
		contest_id := libs.GetIntDefault(ctx, "contest_id", 0)
		_, ok := PRCanSee(ctx, problem_id, contest_id)
		if !ok {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		path := internal.PRGetSampleZip(problem_id)
		_, err := os.Stat(path)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("sample_%d.zip", problem_id))
		}
	}
}

func PRModify(ctx *gin.Context) {
	problem_id, ok := libs.PostInt(ctx, "problem_id")
	if !ok {
		return
	}
	if !internal.PRExists(problem_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, problem_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	title := ctx.PostForm("title")
	if strings.TrimSpace(title) == "" {
		libs.APIWriteBack(ctx, 400, "title cannot be blank", nil)
		return
	}

	pro := internal.PRLoad(problem_id)
	length := len(pro.Statements)
	fix := func(str string) string {
		if len(str) > length {
			return str[:length]
		}
		for i := len(str); i < length; i++ {
			str += "0"
		}
		return str
	}

	var allow_down string
	err := libs.DBSelectSingleColumn(&allow_down, "select allow_down from problems where problem_id=?", problem_id)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	new_allow := ctx.PostForm("allow_down")
	allow_down = fix(allow_down)
	new_allow = fix(new_allow)
	if allow_down != new_allow {
		libs.DBUpdate("update problems set title=?, allow_down=? where problem_id=?", title, new_allow, problem_id)
		err := internal.PRModifySample(problem_id, new_allow)
		if err != nil {
			libs.APIInternalError(ctx, err)
		}
	} else {
		libs.DBUpdate("update problems set title=? where problem_id=?", title, problem_id)
	}
}
