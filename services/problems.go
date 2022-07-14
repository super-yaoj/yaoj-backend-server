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
func PRCanEdit(auth Auth, problem_id int) bool {
	if auth.UserID < 0 {
		return false
	}
	if libs.IsAdmin(auth.UserGrp) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, -auth.UserID)
	return count > 0
}

func PRCanSeeWithoutContest(auth Auth, problem_id int) bool {
	if auth.UserID < 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, libs.DefaultGroup)
		return count > 0
	}
	if PRCanEdit(auth, problem_id) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select * from problem_permissions where problem_id=?) as a join (select * from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", problem_id, auth.UserID)
	return count > 0
}

/*
 */
func PRCanSeeFromContest(auth Auth, problem_id, contest_id int) bool {
	contest, _ := internal.CTQuery(contest_id, auth.UserID)
	if CTCanEnter(auth.UserID, contest, CTCanEdit(auth, contest_id)) &&
		contest.StartTime.Before(time.Now()) && contest.EndTime.After(time.Now()) {
		return internal.CTHasProblem(contest_id, problem_id)
	}
	return false
}

/*
args: contest_id=0 means not in contest

return: (must see from contest, can see)
*/
func PRCanSee(ctx Context, auth Auth, problem_id, contest_id int) (bool, bool) {
	if !PRCanSeeWithoutContest(auth, problem_id) {
		if contest_id <= 0 || !PRCanSeeFromContest(auth, problem_id, contest_id) {
			ctx.JSONAPI(http.StatusForbidden, "", nil)
			return false, false
		}
		return true, true
	}
	return false, true
}

// authorization stored in session
type Auth struct {
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

// pagination query param
type Page struct {
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
}

func (r *Page) CanBound() bool {
	return r.Left != nil || r.Right != nil
}

func (r *Page) Bound() int {
	if r.Left != nil {
		return *r.Left
	} else if r.Right != nil {
		return *r.Right
	}
	return 0
}
func (r *Page) IsLeft() bool {
	return r.Left != nil
}

type ProbListParam struct {
	Auth
	Page
}

func ProbList(ctx Context, param ProbListParam) {
	if !param.Page.CanBound() {
		return
	}
	problems, isfull, err := internal.PRList(
		param.Page.Bound(), param.PageSize, param.UserID, param.IsLeft(), libs.IsAdmin(param.UserGrp),
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
	Auth
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
	CtstID int `query:"contest_id"`
}

func ProbGet(ctx Context, param ProbGetParam) {
	in_contest, ok := PRCanSee(ctx, param.Auth, param.ProbID, param.CtstID)
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
	can_edit := PRCanEdit(param.Auth, param.ProbID)
	if !can_edit {
		prob.DataInfo = problem.DataInfo{}
	}
	ctx.JSONAPI(200, "", map[string]any{"problem": *prob, "can_edit": can_edit})
}

// 获取题目权限
type ProbGetPermParam struct {
	Auth
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
}

func ProbGetPerm(ctx Context, param ProbGetPermParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
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
	ProbID int `body:"problem_id" binding:"required" validate:"probid"`
	PermID int `body:"permission_id" binding:"required"`
	Auth
}

func ProbAddPerm(ctx Context, param ProbAddPermParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
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
	ProbID int `query:"problem_id" binding:"required"`
	PermID int `query:"permission_id" binding:"required"`
	Auth
}

func ProbDelPerm(ctx Context, param ProbDelPermParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbGetMgrParam struct {
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
	Auth
}

func ProbGetMgr(ctx Context, param ProbGetMgrParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
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
	MgrUserID int `body:"user_id" binding:"required"`
	Auth
}

func ProbAddMgr(ctx Context, param ProbAddMgrParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.USExists(param.MgrUserID) {
		ctx.JSONAPI(400, "no such user id", nil)
		return
	}
	err := internal.PRAddPermission(param.ProbID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbDelMgrParam struct {
	ProbID    int `query:"problem_id" binding:"required"`
	MgrUserID int `query:"user_id" binding:"required"`
	Auth
}

func ProbDelMgr(ctx Context, param ProbDelMgrParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.PRDeletePermission(param.ProbID, -param.MgrUserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type ProbPutDataParam struct {
	ProbID int                   `body:"problem_id" binding:"required" validate:"probid"`
	Data   *multipart.FileHeader `body:"data" binding:"required"`
	Auth
}

func ProbPutData(ctx Context, param ProbPutDataParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
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
	Auth
}

func ProbDownData(ctx Context, param ProbDownDataParam) {
	internal.ProblemRWLock.RLock(param.ProbID)
	defer internal.ProblemRWLock.RUnlock(param.ProbID)
	if param.DataType == "data" {
		if !PRCanEdit(param.Auth, param.ProbID) {
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
		_, ok := PRCanSee(ctx, param.Auth, param.ProbID, param.CtstID)
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
	Auth
}

func ProbEdit(ctx Context, param ProbEditParam) {
	if !PRCanEdit(param.Auth, param.ProbID) {
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
