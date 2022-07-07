package services

import (
	"bytes"
	"fmt"
	"io"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
	"github.com/super-yaoj/yaoj-core/pkg/workflow"
)

func SMList(ctx *gin.Context) {
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok {
		return
	}
	problem_id := libs.GetIntDefault(ctx, "problem_id", 0)
	contest_id := libs.GetIntDefault(ctx, "contest_id", 0)
	submitter := libs.GetIntDefault(ctx, "submitter", 0)
	user_id := GetUserId(ctx)

	_, isleft := ctx.GetQuery("left")
	bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	if !ok {
		return
	}
	submissions, isfull, err := internal.SMList(bound, pagesize, user_id, submitter, problem_id, contest_id, isleft, ISAdmin(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	//Modify scores to sample_scores when users are in pretest-only contests
	contest_pretest, err := libs.DBSelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and pretest=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), user_id)
	for key := range submissions {
		if libs.HasIntN(contest_pretest, submissions[key].ContestId) {
			internal.SMPretestOnly(&submissions[key])
		}
	}
	libs.APIWriteBack(ctx, 200, "", map[string]any{"data": submissions, "isfull": isfull})
}

/*
users can submit from a contest if and only if the contest is running and he takes part in the contest
(only these submissions are contest submissions)
*/
func SMSubmit(ctx *gin.Context) {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	problem_id, ok := libs.PostInt(ctx, "problem_id")
	if !ok {
		return
	}
	contest_id := libs.PostIntDefault(ctx, "contest_id", 0)
	in_contest, ok := PRCanSee(ctx, problem_id, contest_id)
	if !ok {
		return
	}
	if !in_contest {
		contest_id = 0
	}

	pro := internal.PRLoad(problem_id)
	if !internal.PRHasData(pro) {
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
	err := internal.SMCreate(user_id, problem_id, contest_id, language, w.Bytes(), preview)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

/*
When the submitted file is a zip file
*/
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
		libs.APIWriteBack(ctx, 400, "invalid zip file: " + err.Error(), nil)
		return nil, nil, 0
	}
	for name, val := range ret {
		matched := ""
		//find corresponding key
		for key := range config {
			if key == name || libs.StartsWith(name, key + ".") {
				matched = key
				break
			}
		}
		if matched == "" {
			libs.APIWriteBack(ctx, 400, "no field matches file name: " + name, nil)
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
			if !ok {
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

/*
Get previews of submitted contents
*/
func getPreview(val []byte, mode utils.CtntType, lang utils.LangTag) internal.ContentPreview {
	const preview_length = 256
	ret := internal.ContentPreview{ Accepted: mode, Language: lang }
	switch mode {
	case utils.Cbinary:
		if len(val) * 2 <= preview_length {
			ret.Content = fmt.Sprintf("%X", val)
		} else {
			ret.Content = fmt.Sprintf("%X", val[: preview_length / 2]) + "..."
		}
	case utils.Cplain:
		if len(val) <= preview_length {
			ret.Content = string(val)
		} else {
			ret.Content = string(val[: preview_length]) + "..."
		}
	case utils.Csource:
		ret.Content = string(val)
	}
	return ret
}

/*
Query single submission, when user is in contests which score_private=true(i.e. cannot see full result),
this function will delete extra information.
*/
func SMQuery(ctx *gin.Context) {
	sid, ok := libs.GetInt(ctx, "submission_id")
	if !ok {
		return
	}
	ret, err := internal.SMQuery(sid)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	//user cannot see submission details inside contests
	no_contest := PRCanSeeWithoutContest(ctx, ret.ProblemId)
	if ret.Submitter != GetUserId(ctx) && !no_contest {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		if !no_contest && internal.CTPretestOnly(ret.ContestId) {
			internal.SMPretestOnly(&ret)
		}
		libs.APIWriteBack(ctx, 200, "", map[string]any{"submission": ret})
	}
}
