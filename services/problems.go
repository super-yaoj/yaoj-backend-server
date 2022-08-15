package services

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"yao/db"
	"yao/internal"

	"github.com/gin-gonic/gin"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	utils "github.com/super-yaoj/yaoj-utils"
)

type ProbListParam struct {
	Auth
	Page `validate:"pagecanbound"`
}

func ProbList(ctx Context, param ProbListParam) {
	problems, isfull, err := internal.ProbList(
		param.Page.Bound(), *param.PageSize, param.UserID, param.IsLeft(), param.IsAdmin(),
	)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"isfull": isfull, "data": problems})
	}
}

type ProbAddParam struct {
	Auth
}

func ProbAdd(ctx Context, param ProbAddParam) {
	param.NewPermit().AsAdmin().Success(func(any) {
		id, err := db.InsertGetId(`insert into problems values (null, "New Problem", 0, "", "")`)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", gin.H{"id": id})
		}
	}).FailAPIStatusForbidden(ctx)
}

// 查询问题
type ProbGetParam struct {
	Auth
	ProbID int `query:"problem_id" validate:"required,probid"`
	CtstID int `query:"contest_id"`
}

func ProbGet(ctx Context, param ProbGetParam) {
	param.NewPermit().TrySeeProb(param.ProbID, param.CtstID).Success(func(a any) {
		ctstid := a.(int)
		prob, err := internal.ProbQuery(param.ProbID, param.UserID)
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
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"problem": ret_prob, "can_edit": can_edit, "in_contest": ctstid})
	}).FailAPIStatusForbidden(ctx)
}

// 获取题目权限
type ProbGetPermParam struct {
	Auth
	ProbID int `query:"problem_id" validate:"required,probid"`
}

func ProbGetPerm(ctx Context, param ProbGetPermParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		pers, err := internal.ProbGetPermissions(param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": pers})
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbAddPermParam struct {
	Auth
	ProbID int `body:"problem_id" validate:"required,probid"`
	PermID int `body:"permission_id" validate:"required,prmsid"`
}

func ProbAddPerm(ctx Context, param ProbAddPermParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		err := internal.ProbAddPermission(param.ProbID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbDelPermParam struct {
	Auth
	ProbID int `query:"problem_id" validate:"required,probid"`
	PermID int `query:"permission_id" validate:"required,prmsid"`
}

func ProbDelPerm(ctx Context, param ProbDelPermParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		err := internal.ProbDeletePermission(param.ProbID, param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbGetMgrParam struct {
	Auth
	ProbID int `query:"problem_id" validate:"required,probid"`
}

func ProbGetMgr(ctx Context, param ProbGetMgrParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		users, err := internal.ProbGetManagers(param.ProbID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbAddMgrParam struct {
	Auth
	ProbID    int `body:"problem_id" validate:"required,probid"`
	MgrUserID int `body:"user_id" validate:"required,userid"`
}

func ProbAddMgr(ctx Context, param ProbAddMgrParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		err := internal.ProbAddPermission(param.ProbID, -param.MgrUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbDelMgrParam struct {
	Auth
	ProbID    int `query:"problem_id" bvalidate:"required,probid"`
	MgrUserID int `query:"user_id" validate:"required,userid"`
}

func ProbDelMgr(ctx Context, param ProbDelMgrParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		err := internal.ProbDeletePermission(param.ProbID, -param.MgrUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbPutDataParam struct {
	Auth
	ProbID int                   `body:"problem_id" validate:"required,probid"`
	Data   *multipart.FileHeader `body:"data" validate:"required"`
}

func ProbPutData(ctx Context, param ProbPutDataParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		ext := path.Ext(param.Data.Filename)
		if ext != ".zip" {
			ctx.JSONAPI(http.StatusBadRequest, "unsupported file extension "+ext, nil)
			return
		}
		tmpdir := utils.GetTempDir()
		defer os.RemoveAll(tmpdir)
		err := ctx.SaveUploadedFile(param.Data, path.Join(tmpdir, "1.zip"))
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
		log.Printf("file uploaded saves at %s", path.Join(tmpdir, "1.zip"))
		err = internal.ProbPutData(param.ProbID, tmpdir)
		if err != nil {
			ctx.JSONAPI(http.StatusBadRequest, err.Error(), nil)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbDownDataParam struct {
	Auth
	ProbID   int    `query:"problem_id" validate:"required,probid"`
	CtstID   int    `query:"contest_id"`
	DataType string `query:"type" validate:"required"`
}

func ProbDownData(ctx Context, param ProbDownDataParam) {
	internal.ProblemRWLock.RLock(param.ProbID)
	defer internal.ProblemRWLock.RUnlock(param.ProbID)
	if param.DataType == "data" {
		param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
			path := internal.ProbGetDataZip(param.ProbID)
			_, err := os.Stat(path)
			if err != nil {
				ctx.JSONAPI(http.StatusBadRequest, "no data", nil)
			} else {
				ctx.FileAttachment(path, fmt.Sprintf("problem_%d.zip", param.ProbID))
			}
		}).FailAPIStatusForbidden(ctx)
	} else {
		param.NewPermit().TrySeeProb(param.ProbID, param.CtstID).Success(func(any) {
			path := internal.ProbGetSampleZip(param.ProbID)
			_, err := os.Stat(path)
			if err != nil {
				ctx.JSONAPI(http.StatusBadRequest, "no data", nil)
			} else {
				ctx.FileAttachment(path, fmt.Sprintf("sample_%d.zip", param.ProbID))
			}
		}).FailAPIStatusForbidden(ctx)
	}
}

type ProbEditParam struct {
	Auth
	ProbID    int    `body:"problem_id" validate:"required,probid"`
	Title     string `body:"title" validate:"required,gte=1,lte=190"`
	AllowDown string `body:"allow_down"`
}

func ProbEdit(ctx Context, param ProbEditParam) {
	param.NewPermit().TryEditProb(param.ProbID).Success(func(any) {
		if strings.TrimSpace(param.Title) == "" {
			ctx.JSONAPI(http.StatusBadRequest, "title cannot be blank", nil)
			return
		}

		pro := internal.ProbLoad(param.ProbID)
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
		err := db.SelectSingleColumn(&allow_down, "select allow_down from problems where problem_id=?", param.ProbID)
		if err != nil {
			ctx.JSONAPI(http.StatusNotFound, "", nil)
			return
		}
		allow_down = fix(allow_down)
		new_allow := fix(param.AllowDown)
		if allow_down != new_allow {
			db.Update("update problems set title=?, allow_down=? where problem_id=?", param.Title, new_allow, param.ProbID)
			err := internal.ProbModifySample(param.ProbID, new_allow)
			if err != nil {
				ctx.ErrorAPI(err)
			}
		} else {
			db.Update("update problems set title=? where problem_id=?", param.Title, param.ProbID)
		}
	}).FailAPIStatusForbidden(ctx)
}

type ProbStatisticParam struct {
	Auth
	Page    `validate:"pagecanbound"`
	ProbID  int    `query:"problem_id" validate:"required,probid"`
	Mode    string `query:"mode" validate:"required"` //one of {"time", "memory"}
	LeftId  *int   `query:"left_id"`
	RightId *int   `query:"right_id"`
}

func ProbStatistic(ctx Context, param ProbStatisticParam) {
	//Users can see statistics if and only if they are out of contest
	param.NewPermit().TrySeeProb(param.ProbID, 0).Success(func(any) {
		subs, isfull := []int{}, false
		if param.IsLeft() {
			if param.LeftId == nil {
				ctx.JSONAPI(http.StatusBadRequest, "left_id is null", nil)
				return
			}
			subs, isfull = internal.ProbSGetSubmissions(param.ProbID, *param.Left, *param.LeftId, *param.PageSize, true, param.Mode)
		} else {
			if param.RightId == nil {
				ctx.JSONAPI(http.StatusBadRequest, "right_id is null", nil)
				return
			}
			subs, isfull = internal.ProbSGetSubmissions(param.ProbID, *param.Right, *param.RightId, *param.PageSize, false, param.Mode)
		}
		if subs == nil {
			ctx.JSONAPI(http.StatusBadRequest, "", nil)
			return
		}
		ret := internal.SubmListByIds(subs)
		acnum, totnum := internal.ProbSGetACRatio(param.ProbID)
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": ret, "isfull": isfull, "acnum": acnum, "totnum": totnum})
	}).FailAPIStatusForbidden(ctx)
}
