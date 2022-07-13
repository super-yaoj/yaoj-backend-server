package services

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"yao/internal"
	"yao/libs"
	"yao/service"

	"github.com/gin-gonic/gin"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
)

// i. e. valid user_id with either:
// 1. admin user_group
// 2. problem permission
func PRCanEdit(user_id int, user_group int, problem_id int) bool {
	if user_id < 0 {
		return false
	}
	if libs.IsAdmin(user_group) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, -user_id)
	return count > 0
}

func PRCanSeeWithoutContest(user_id int, user_group int, problem_id int) bool {
	if user_id < 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, libs.DefaultGroup)
		return count > 0
	}
	if PRCanEdit(user_id, user_group, problem_id) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select * from problem_permissions where problem_id=?) as a join (select * from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", problem_id, user_id)
	return count > 0
}

/*
 */
func PRCanSeeFromContest(user_id, user_group, problem_id, contest_id int) bool {
	contest, _ := internal.CTQuery(contest_id, user_id)
	if CTCanEnter(user_id, contest, CTCanEdit(user_id, user_group, contest_id)) &&
		contest.StartTime.Before(time.Now()) && contest.EndTime.After(time.Now()) {
		return internal.CTHasProblem(contest_id, problem_id)
	}
	return false
}

/*
args: contest_id=0 means not in contest

return: (must see from contest, can see)
*/
func PRCanSee(ctx Context, user_id, user_group, problem_id, contest_id int) (bool, bool) {
	if !PRCanSeeWithoutContest(user_id, user_group, problem_id) {
		if contest_id <= 0 || !PRCanSeeFromContest(user_id, user_group, problem_id, contest_id) {
			ctx.JSONAPI(http.StatusForbidden, "", nil)
			return false, false
		}
		return true, true
	}
	return false, true
}

type ProbListParam struct {
	UserID   int  `session:"user_id"`
	UserGrp  int  `session:"user_group"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
}

func ProbList(ctx Context, param ProbListParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	problems, isfull, err := internal.PRList(
		bound, param.PageSize, param.UserID, param.Left != nil, libs.IsAdmin(param.UserGrp),
	)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"isfull": isfull, "data": problems})
	}
}

type ProbAddParam struct {
	UserGrp int `session:"user_group" validate:"admin"`
}

func ProbAdd(ctx Context, param ProbAddParam) {
	id, err := libs.DBInsertGetId(`insert into problems values (null, "New Problem", 0, "", "")`)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"id": id})
	}
}

// 查询问题
type ProbGetParam struct {
	ProbID  int `query:"problem_id" binding:"required" validate:"probid"`
	CtstID  int `query:"contest_id"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbGet(ctx Context, param ProbGetParam) {
	in_contest, ok := PRCanSee(ctx, param.UserID, param.UserGrp, param.ProbID, param.CtstID)
	if !ok {
		return
	}
	prob, err := internal.PRQuery(param.ProbID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
	if in_contest {
		prob.Tutorial_zh = ""
		prob.Tutorial_en = ""
	}
	can_edit := PRCanEdit(param.UserID, param.UserGrp, param.ProbID)
	if !can_edit {
		prob.DataInfo = problem.DataInfo{}
	}
	ctx.JSONAPI(200, "", map[string]any{"problem": *prob, "can_edit": can_edit})
}

// 获取题目权限
type ProbGetPermParam struct {
	ProbID  int `query:"problem_id" binding:"required" validate:"probid"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbGetPerm(ctx Context, param ProbGetPermParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	pers, err := internal.PRGetPermissions(param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": pers})
	}
}

type ProbAddPermParam struct {
	ProbID  int `body:"problem_id" binding:"required" validate:"probid"`
	PermID  int `body:"permission_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbAddPerm(ctx Context, param ProbAddPermParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		ctx.JSONAPI(400, "no such permission id", nil)
		return
	}
	err := internal.PRAddPermission(param.ProbID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbDelPermParam struct {
	ProbID  int `query:"problem_id" binding:"required"`
	PermID  int `query:"permission_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbDelPerm(ctx Context, param ProbDelPermParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbGetMgrParam struct {
	ProbID  int `query:"problem_id" binding:"required" validate:"probid"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func ProbGetMgr(ctx Context, param ProbGetMgrParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	users, err := internal.PRGetManagers(param.ProbID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": users})
	}
}

type ProbAddMgrParam struct {
	ProbID    int `body:"problem_id" binding:"required" validate:"probid"`
	UserID    int `body:"user_id" binding:"required"`
	CurUserID int `session:"user_id"`
	UserGrp   int `session:"user_group"`
}

func ProbAddMgr(ctx Context, param ProbAddMgrParam) {
	if !PRCanEdit(param.CurUserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.USExists(param.UserID) {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	}
	err := internal.PRAddPermission(param.ProbID, -param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbDelMgrParam struct {
	ProbID    int `query:"problem_id" binding:"required"`
	UserID    int `query:"user_id" binding:"required"`
	CurUserID int `session:"user_id"`
	UserGrp   int `session:"user_group"`
}

func ProbDelMgr(ctx Context, param ProbDelMgrParam) {
	if !PRCanEdit(param.CurUserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, -param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbPutDataParam struct {
	ProbID  int                   `body:"problem_id" binding:"required" validate:"probid"`
	Data    *multipart.FileHeader `body:"data" binding:"required"`
	UserID  int                   `session:"user_id"`
	UserGrp int                   `session:"user_group"`
}

func ProbPutData(ctx Context, param ProbPutDataParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	ext := path.Ext(param.Data.Filename)
	if ext != ".zip" {
		ctx.JSONAPI(400, "unsupported file extension "+ext, nil)
		return
	}

	tmpdir := libs.GetTempDir()
	defer os.RemoveAll(tmpdir)
	err := ctx.SaveUploadedFile(param.Data, path.Join(tmpdir, "1.zip"))
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
	log.Printf("file uploaded saves at %s", path.Join(tmpdir, "1.zip"))
	err = internal.PRPutData(param.ProbID, tmpdir)
	if err != nil {
		ctx.JSONAPI(400, err.Error(), nil)
	}
}

type ProbDownDataParam struct {
	ProbID   int    `query:"problem_id" binding:"required" validate:"probid"`
	CtstID   int    `query:"contest_id"`
	DataType string `query:"type"`
	UserID   int    `session:"user_id"`
	UserGrp  int    `session:"user_group"`
}

func ProbDownData(ctx Context, param ProbDownDataParam) {
	internal.ProblemRWLock.RLock(param.ProbID)
	defer internal.ProblemRWLock.RUnlock(param.ProbID)
	if param.DataType == "data" {
		if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
			ctx.JSONAPI(403, "", nil)
			return
		}
		path := internal.PRGetDataZip(param.ProbID)
		_, err := os.Stat(path)
		if err != nil {
			ctx.JSONAPI(400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("problem_%d.zip", param.ProbID))
		}
	} else {
		_, ok := PRCanSee(ctx, param.UserID, param.UserGrp, param.ProbID, param.CtstID)
		if !ok {
			ctx.JSONAPI(403, "", nil)
			return
		}
		path := internal.PRGetSampleZip(param.ProbID)
		_, err := os.Stat(path)
		if err != nil {
			ctx.JSONAPI(400, "no data", nil)
		} else {
			ctx.FileAttachment(path, fmt.Sprintf("sample_%d.zip", param.ProbID))
		}
	}
}

type Context = service.Context

type ProbEditParam struct {
	ProbID    int    `body:"problem_id" binding:"required" validate:"probid"`
	Title     string `body:"title"`
	AllowDown string `body:"allow_down"`
	UserID    int    `session:"user_id"`
	UserGrp   int    `session:"user_group"`
}

func ProbEdit(ctx Context, param ProbEditParam) {
	if !PRCanEdit(param.UserID, param.UserGrp, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if strings.TrimSpace(param.Title) == "" {
		ctx.JSONAPI(400, "title cannot be blank", nil)
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
		ctx.JSONAPI(404, "", nil)
		return
	}
	allow_down = fix(allow_down)
	new_allow := fix(param.AllowDown)
	if allow_down != new_allow {
		libs.DBUpdate("update problems set title=?, allow_down=? where problem_id=?", param.Title, new_allow, param.ProbID)
		err := internal.PRModifySample(param.ProbID, new_allow)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	} else {
		libs.DBUpdate("update problems set title=? where problem_id=?", param.Title, param.ProbID)
	}
}
