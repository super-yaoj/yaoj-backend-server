package services

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"yao/internal"
	"yao/libs"
	"yao/service"

	"github.com/gin-gonic/gin"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
)

type ProbListParam struct {
	Auth
	Page
}

func ProbList(ctx Context, param ProbListParam) {
	if !param.Page.CanBound() {
		return
	}
	problems, isfull, err := internal.PRList(
		param.Page.Bound(), param.PageSize, param.UserID, param.IsLeft(), param.IsAdmin(),
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
	param.Auth.SetCtst(param.CtstID).TrySeeProb(param.ProbID).
		Then(func(ctstid int) {
			prob, err := internal.PRQuery(param.ProbID, param.UserID)
			if err != nil {
				ctx.ErrorAPI(err)
				return
			}
			ret_prob := *prob
			if ctstid > 0 {
				ret_prob.Tutorial_zh = ""
				ret_prob.Tutorial_en = ""
			}
			can_edit := param.Auth.CanEditProb(param.ProbID)
			if !can_edit {
				ret_prob.DataInfo = problem.DataInfo{}
			}
			ctx.JSONAPI(200, "", map[string]any{"problem": ret_prob, "can_edit": can_edit, "in_contest": ctstid})
		}).Else(func(ctstid int) {
			ctx.JSONAPI(http.StatusForbidden, "", nil)
		})
}

// 获取题目权限
type ProbGetPermParam struct {
	Auth
	ProbID int `query:"problem_id" binding:"required" validate:"probid"`
}

func ProbGetPerm(ctx Context, param ProbGetPermParam) {
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
	if !param.Auth.CanEditProb(param.ProbID) {
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
		if !param.Auth.CanEditProb(param.ProbID) {
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
		param.Auth.SetCtst(param.CtstID).TrySeeProb(param.ProbID).
			Then(func(ctstid int) {
				path := internal.PRGetSampleZip(param.ProbID)
				_, err := os.Stat(path)
				if err != nil {
					ctx.JSONAPI(400, "no data", nil)
				} else {
					ctx.FileAttachment(path, fmt.Sprintf("sample_%d.zip", param.ProbID))
				}
			}).
			Else(func(a int) {
				ctx.JSONAPI(403, "", nil)
			})
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
	if !param.Auth.CanEditProb(param.ProbID) {
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

type ProbStatisticParam struct {
	Auth
	Page
	ProbID    int    `query:"problem_id" binding:"required" validate:"probid"`
	Mode      string `query:"mode" binding:"required"`//one of {"time", "memory"}
	LeftId   *int    `query:"left_id"`
	RightId  *int    `query:"right_id"`
}

func ProbStatistic(ctx Context, param ProbStatisticParam) {
	//Users can see statistics if and only if they are out of contest
	param.TrySeeProb(param.ProbID).Then(func(ctstid int) {
		if !param.CanBound() {
			ctx.JSONAPI(400, "both left and right are null", nil)
			return
		}
		subs, isfull := []int{}, false
		if param.IsLeft() {
			if param.LeftId == nil {
				ctx.JSONAPI(400, "left_id is null", nil)
				return
			}
			subs, isfull = internal.PRSGetSubmissions(param.ProbID, *param.Left, *param.LeftId, param.PageSize, true, param.Mode)
		} else {
			if param.RightId == nil {
				ctx.JSONAPI(400, "right_id is null", nil)
				return
			}
			subs, isfull = internal.PRSGetSubmissions(param.ProbID, *param.Right, *param.RightId, param.PageSize, false, param.Mode)
		}
		if subs == nil {
			ctx.JSONAPI(400, "", nil)
			return
		}
		ret := internal.SMListByIds(subs)
		acnum, totnum := internal.PRSGetACRatio(param.ProbID)
		ctx.JSONAPI(200, "", map[string]any{"data": ret, "isfull": isfull, "acnum": acnum, "totnum": totnum})
	}).Else(func(ctstid int) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
	})
}