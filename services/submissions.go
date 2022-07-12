package services

import (
	"bytes"
	"fmt"
	"io"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
	"github.com/super-yaoj/yaoj-core/pkg/workflow"
)

type SubmListParam struct {
	UserID   int  `session:"user_id"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
	ProbID   int  `query:"problem_id"`
	CtstID   int  `query:"contest_id"`
	Submtter int  `query:"submitter"`
}

func SubmList(ctx *gin.Context, param SubmListParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	submissions, isfull, err := internal.SMList(
		bound, param.PageSize, param.UserID, param.Submtter, param.ProbID, param.CtstID,
		param.Left != nil, ISAdmin(ctx),
	)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	//Modify scores to sample_scores when users are in pretest-only contests
	contest_pretest, _ := libs.DBSelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and pretest=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), param.UserID)
	for key := range submissions {
		if libs.HasElement(contest_pretest, submissions[key].ContestId) {
			internal.SMPretestOnly(&submissions[key])
		}
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"data": submissions, "isfull": isfull})
}

type SubmAddParam struct {
	UserID int `session:"user_id"`
	ProbID int `body:"problem_id" binding:"required"`
	CtstID int `body:"contest_id"`
}

// users can submit from a contest if and only if the contest is running and
// he takes part in the contest (only these submissions are contest submissions)
func SubmAdd(ctx *gin.Context, param SubmAddParam) {
	if param.UserID <= 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	in_contest, ok := PRCanSee(ctx, param.ProbID, param.CtstID)
	if !ok {
		return
	}
	if !in_contest {
		param.CtstID = 0
	}

	pro := internal.PRLoad(param.ProbID)
	if !internal.PRHasData(pro, "tests") {
		libs.APIWriteBack(ctx, 400, "problem has no data", nil)
		return
	}
	var sub problem.Submission
	var language utils.LangTag
	var preview map[string]internal.ContentPreview
	_, all := ctx.GetPostForm("submit_all")
	if all {
		sub, preview, language = parseZipFile(ctx, "all.zip", pro.SubmConfig)
	} else {
		sub, preview, language = parseMultiFiles(ctx, pro.SubmConfig)
	}
	if sub == nil {
		return
	}

	w := bytes.NewBuffer(nil)
	sub.DumpTo(w)
	err := internal.SMCreate(param.UserID, param.ProbID, param.CtstID, language, w.Bytes(), preview)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

// When the submitted file is a zip file
func parseZipFile(ctx *gin.Context, field string, config internal.SubmConfig) (problem.Submission, map[string]internal.ContentPreview, utils.LangTag) {
	file, header, err := ctx.Request.FormFile(field)
	if err != nil {
		return nil, nil, 0
	}
	language := -1
	sub := make(problem.Submission)
	preview := make(map[string]internal.ContentPreview)

	var tot_size int64 = 0
	for _, val := range config {
		tot_size += int64(val.Length)
	}
	if header.Size > tot_size {
		libs.APIWriteBack(ctx, 400, "file too large", nil)
		return nil, nil, 0
	}

	w := bytes.NewBuffer(nil)
	_, err = io.Copy(w, file)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return nil, nil, 0
	}
	ret, err := libs.UnzipMemory(w.Bytes())
	if err != nil {
		libs.APIWriteBack(ctx, 400, "invalid zip file: "+err.Error(), nil)
		return nil, nil, 0
	}
	for name, val := range ret {
		matched := ""
		//find corresponding key
		for key := range config {
			if key == name || libs.StartsWith(name, key+".") {
				matched = key
				break
			}
		}
		if matched == "" {
			libs.APIWriteBack(ctx, 400, "no field matches file name: "+name, nil)
			return nil, nil, 0
		}
		//TODO: get language by file suffix
		preview[matched] = getPreview(val, config[matched].Accepted, -1)
		sub.SetSource(workflow.Gsubm, matched, name, bytes.NewReader(val))
	}
	return sub, preview, utils.LangTag(language)
}

/*
When user submits files one by one
*/
func parseMultiFiles(ctx *gin.Context, config internal.SubmConfig) (problem.Submission, map[string]internal.ContentPreview, utils.LangTag) {
	sub := make(problem.Submission)
	preview := make(map[string]internal.ContentPreview)
	language := -1
	for key, val := range config {
		str, ok := ctx.GetPostForm(key + "_text")
		name, lang := key, -1
		//get language
		if val.Accepted == utils.Csource {
			lang, ok = libs.PostIntRange(ctx, key+"_lang", 0, len(libs.LangSuf)-1)
			if !ok || (val.Langs != nil && !libs.HasElement(val.Langs, utils.LangTag(lang))) {
				return nil, nil, 0
			}
			language = lang
			name += libs.LangSuf[lang]
		}
		if ok {
			//text
			if len(str) > int(val.Length) {
				libs.APIWriteBack(ctx, 400, "file "+key+" too large", nil)
				return nil, nil, 0
			}
			byt := []byte(str)
			preview[key] = getPreview(byt, val.Accepted, utils.LangTag(lang))
			sub.SetSource(workflow.Gsubm, key, name, bytes.NewReader(byt))
		} else {
			//file
			file, header, err := ctx.Request.FormFile(key + "_file")
			if err != nil {
				libs.APIInternalError(ctx, err)
				return nil, nil, 0
			}
			if header.Size > int64(val.Length) {
				libs.APIWriteBack(ctx, 400, "file "+key+" too large", nil)
				return nil, nil, 0
			}
			w := bytes.NewBuffer(nil)
			_, err = io.Copy(w, file)
			if err != nil {
				libs.APIInternalError(ctx, err)
				return nil, nil, 0
			}
			preview[key] = getPreview(w.Bytes(), val.Accepted, utils.LangTag(lang))
			sub.SetSource(workflow.Gsubm, key, name, w)
		}
	}
	return sub, preview, utils.LangTag(language)
}

// Get previews of submitted contents
func getPreview(val []byte, mode utils.CtntType, lang utils.LangTag) internal.ContentPreview {
	const preview_length = 256
	ret := internal.ContentPreview{Accepted: mode, Language: lang}
	switch mode {
	case utils.Cbinary:
		if len(val)*2 <= preview_length {
			ret.Content = fmt.Sprintf("%X", val)
		} else {
			ret.Content = fmt.Sprintf("%X", val[:preview_length/2]) + "..."
		}
	case utils.Cplain:
		if len(val) <= preview_length {
			ret.Content = string(val)
		} else {
			ret.Content = string(val[:preview_length]) + "..."
		}
	case utils.Csource:
		ret.Content = string(val)
	}
	return ret
}

type SubmGetParam struct {
	SubmID int `query:"submission_id" binding:"required"`
	UserID int `session:"user_id"`
}

// Query single submission, when user is in contests which score_private=true
// (i.e. cannot see full result), this function will delete extra information.
func SubmGet(ctx *gin.Context, param SubmGetParam) {
	ret, err := internal.SMQuery(param.SubmID)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	//user cannot see submission details inside contests
	by_problem := PRCanSeeWithoutContest(ctx, ret.ProblemId)
	can_edit := SMCanEdit(ctx, ret.SubmissionBase)
	if !can_edit && ret.Submitter != param.UserID && !by_problem {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		if !can_edit && !by_problem {
			if internal.CTPretestOnly(ret.ContestId) {
				internal.SMPretestOnly(&ret)
			} else {
				ret.Details.Result = internal.SMRemoveTestDetails(ret.Details.Result)
				ret.Details.ExtraResult = internal.SMRemoveTestDetails(ret.Details.ExtraResult)
			}
		}
		libs.APIWriteBack(ctx, 200, "", map[string]any{"submission": ret, "can_edit": can_edit})
	}
}

func SMCustomTest(ctx *gin.Context) {
	config := internal.SubmConfig{
		"source": {Langs: nil, Accepted: utils.Csource, Length: 64 * 1024},
		"input":  {Langs: nil, Accepted: utils.Cplain, Length: 10 * 1024 * 1024},
	}
	subm, _, _ := parseMultiFiles(ctx, config)
	if subm == nil {
		return
	}
	w := bytes.NewBuffer(nil)
	subm.DumpTo(w)
	result := internal.SMJudgeCustomTest(w.Bytes())
	if len(result) == 0 {
		result, _ = json.Marshal(map[string]any{
			"Memory": -1,
			"Time":   -1,
			"Title":  "Internal Error",
		})
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"result": string(result)})
}

func SMCanEdit(ctx *gin.Context, sub internal.SubmissionBase) bool {
	return PRCanEdit(ctx, sub.ProblemId) || (sub.ContestId > 0 && CTCanEdit(ctx, sub.ContestId))
}

type SubmDelParam struct {
	SubmID int `query:"submission_id" binding:"required"`
}

func SubmDel(ctx *gin.Context, param SubmDelParam) {
	sub, err := internal.SMGetBaseInfo(param.SubmID)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !SMCanEdit(ctx, sub) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err = internal.SMDelete(sub)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type RejudgeParam struct {
	ProbID *int `body:"problem_id"`
	SubmID int  `body:"submission_id"`
}

func Rejudge(ctx *gin.Context, param RejudgeParam) {
	if param.ProbID != nil {
		if !internal.PRExists(*param.ProbID) {
			libs.RPCWriteBack(ctx, 404, -32600, "", nil)
			return
		}
		if !PRCanEdit(ctx, *param.ProbID) {
			libs.RPCWriteBack(ctx, 403, -32600, "", nil)
			return
		}
		err := internal.PRRejudge(*param.ProbID)
		if err != nil {
			libs.RPCInternalError(ctx, err)
		}
	} else {
		sub, err := internal.SMGetBaseInfo(param.SubmID)
		if err != nil {
			libs.RPCWriteBack(ctx, 404, -32600, "", nil)
			return
		}
		if !SMCanEdit(ctx, sub) {
			libs.RPCWriteBack(ctx, 403, -32600, "", nil)
			return
		}
		err = internal.SMRejudge(param.SubmID)
		if err != nil {
			libs.RPCInternalError(ctx, err)
		}
	}
}
