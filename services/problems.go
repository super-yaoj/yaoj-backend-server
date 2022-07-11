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

type ProbListParam struct {
	UserID   int  `session:"user_id"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
}

func ProbList(ctx *gin.Context, param ProbListParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	problems, isfull, err := internal.PRList(bound, param.PageSize, param.UserID, param.Left != nil, ISAdmin(ctx))
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
	ProbID  int `query:"problem_id" binding:"required"`
	CtstID  int `query:"contest_id"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbGet(ctx *gin.Context, param ProbGetParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	in_contest, ok := PRCanSee(ctx, param.ProbID, param.CtstID)
	if !ok {
		return
	}
	prob, err := internal.PRQuery(param.ProbID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	if in_contest {
		prob.Tutorial_zh = ""
		prob.Tutorial_en = ""
	}
	can_edit := PRCanEdit(ctx, param.ProbID)
	if !can_edit {
		prob.DataInfo = problem.DataInfo{}
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"problem": *prob, "can_edit": can_edit})
}

// 获取题目权限
type ProbGetPermParam struct {
	ProbID int `query:"problem_id" binding:"required"`
}

func ProbGetPerm(ctx *gin.Context, param ProbGetPermParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	pers, err := internal.PRGetPermissions(param.ProbID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": pers})
	}
}

type ProbAddPermParam struct {
	ProbID int `body:"problem_id" binding:"required"`
	PermID int `body:"permission_id" binding:"required"`
}

func ProbAddPerm(ctx *gin.Context, param ProbAddPermParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := internal.PRAddPermission(param.ProbID, param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type ProbDelPermParam struct {
	ProbID int `query:"problem_id" binding:"required"`
	PermID int `query:"permission_id" binding:"required"`
}

func ProbDelPerm(ctx *gin.Context, param ProbDelPermParam) {
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type ProbGetMgrParam struct {
	ProbID int `query:"problem_id" binding:"required"`
}

func ProbGetMgr(ctx *gin.Context, param ProbGetMgrParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	users, err := internal.PRGetManagers(param.ProbID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users})
	}
}

type ProbAddMgrParam struct {
	ProbID int `body:"problem_id" binding:"required"`
	UserID int `body:"user_id" binding:"required"`
}

func ProbAddMgr(ctx *gin.Context, param ProbAddMgrParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if !internal.USExists(param.UserID) {
		libs.APIWriteBack(ctx, 400, "no such user id", nil)
		return
	}
	err := internal.PRAddPermission(param.ProbID, -param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type ProbDelMgrParam struct {
	ProbID int `query:"problem_id" binding:"required"`
	UserID int `query:"user_id" binding:"required"`
}

func ProbDelMgr(ctx *gin.Context, param ProbDelMgrParam) {
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, -param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type ProbPutDataParam struct {
	ProbID int `body:"problem_id" binding:"required"`
}

func ProbPutData(ctx *gin.Context, param ProbPutDataParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	file, err := ctx.FormFile("data")
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
		return
	}
	ext := path.Ext(file.Filename)
	if ext != ".zip" {
		libs.APIWriteBack(ctx, 400, "doesn't support file extension "+ext, nil)
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
	err = internal.PRPutData(param.ProbID, tmpdir)
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
	}
}

type ProbDownDataParam struct {
	ProbID   int    `query:"problem_id" binding:"required"`
	CtstID   int    `query:"contest_id"`
	DataType string `query:"type"`
}

func ProbDownData(ctx *gin.Context, param ProbDownDataParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	internal.ProblemRWLock.RLock(param.ProbID)
	defer internal.ProblemRWLock.RUnlock(param.ProbID)
	if param.DataType == "data" {
		if !PRCanEdit(ctx, param.ProbID) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		path := internal.PRGetDataZip(param.ProbID)
		_, err := os.Stat(path)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("problem_%d.zip", param.ProbID))
		}
	} else {
		_, ok := PRCanSee(ctx, param.ProbID, param.CtstID)
		if !ok {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		path := internal.PRGetSampleZip(param.ProbID)
		_, err := os.Stat(path)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("sample_%d.zip", param.ProbID))
		}
	}
}

type ProbModifyParam struct {
	ProbID    int    `body:"problem_id" binding:"required"`
	Title     string `body:"title"`
	AllowDown string `body:"allow_down"`
}

func ProbModify(ctx *gin.Context, param ProbModifyParam) {
	if !internal.PRExists(param.ProbID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanEdit(ctx, param.ProbID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if strings.TrimSpace(param.Title) == "" {
		libs.APIWriteBack(ctx, 400, "title cannot be blank", nil)
		return
	}

	pro := internal.PRLoad(param.ProbID)
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
	err := libs.DBSelectSingleColumn(&allow_down, "select allow_down from problems where problem_id=?", param.ProbID)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	allow_down = fix(allow_down)
	new_allow := fix(param.AllowDown)
	if allow_down != new_allow {
		libs.DBUpdate("update problems set title=?, allow_down=? where problem_id=?", param.Title, new_allow, param.ProbID)
		err := internal.PRModifySample(param.ProbID, new_allow)
		if err != nil {
			libs.APIInternalError(ctx, err)
		}
	} else {
		libs.DBUpdate("update problems set title=? where problem_id=?", param.Title, param.ProbID)
	}
}
