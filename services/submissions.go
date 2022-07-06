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
	columns := "submission_id, submitter, problem_id, contest_id, status, score, time, memory, language, submit_time"

	_, isleft := ctx.GetQuery("left")
	bound, _ := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	query := libs.If(problem_id == 0, "", fmt.Sprintf(" and problem_id=%d", problem_id)) +
		libs.If(contest_id == 0, "", fmt.Sprintf(" and contest_id=%d", contest_id)) +
		libs.If(submitter == 0, "", fmt.Sprintf(" and submitter=%d", submitter))
	must := "1"
	if !ISAdmin(ctx) {
		perms, _ := internal.USPermissions(user_id)
		perm_str := libs.JoinArray(perms)
		//problems user can see
		probs, _ := libs.DBSelectInts("select problem_id from problem_permissions where permission_id in (" + perm_str + ")")
		/*
			First, user can see all finished contests
			For running contests, participants cannnot see other's contest submissions if score_private=1
			For not started contests, they must contain no contest submissions according to the definition, so we can discard them
		*/
		conts, _ := libs.DBSelectInts("select contest_id from contest_permissions where permission_id in (" + perm_str + ")")
		//remove contests that cannot see
		conts_running, _ := libs.DBSelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and score_private=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), user_id)
		for i := range conts {
			//running contests is few, so brute force is just ok
			if libs.HasIntN(conts_running, conts[i]) {
				conts[i] = 0
			}
		}

		must = "("
		if problem_id == 0 {
			must += libs.If(len(probs) == 0, "0", "(problem_id in (" + libs.JoinArray(probs) + "))")
		} else {
			must += libs.If(libs.HasIntN(probs, problem_id), "1", "0")
		}
		if contest_id == 0 {
			must += libs.If(len(conts) == 0, "0", " or (contest_id in (" + libs.JoinArray(conts) + "))")
		} else {
			must += " or " + libs.If(libs.HasIntN(conts, contest_id), "1", "0")
		}
		if submitter == 0 {
			if user_id > 0 {
				must += fmt.Sprintf(" or submitter=%d)", user_id)
			}
		} else {
			must += libs.If(submitter == user_id, " or 1)", ")")
		}
	}
	pagesize += 1
	var submissions []internal.Submission
	if isleft {
		libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id<=%d and %s %s order by submission_id desc limit %d", columns, bound, must, query, pagesize))
	} else {
		libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id>=%d and %s %s order by submission_id limit %d", columns, bound, must, query, pagesize))
	}
	isfull := len(submissions) == pagesize
	if isfull {
		submissions = submissions[:pagesize-1]
	}
	if !isleft {
		libs.Reverse(submissions)
	}
	internal.SMGetExtraInfo(submissions)
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
	if pro.Id != problem_id {
		libs.APIWriteBack(ctx, 400, "problem has no data", nil)
		return
	}
	var sub problem.Submission
	var language utils.LangTag
	var preview map[string]string
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
func parseZipFile(ctx *gin.Context, field string, config internal.SubmConfig) (problem.Submission, map[string]string, utils.LangTag) {
	file, header, err := ctx.Request.FormFile(field)
	if err != nil {
		return nil, nil, 0
	}
	language := -1
	sub := make(problem.Submission)
	preview := make(map[string]string)

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
		preview[matched] = getPreview(val, config[matched].Accepted)
		sub.SetSource(workflow.Gsubm, matched, name, bytes.NewReader(val))
	}
	return sub, preview, utils.LangTag(language)
}

/*
When user submits files one by one
*/
func parseMultiFiles(ctx *gin.Context, config internal.SubmConfig) (problem.Submission, map[string]string, utils.LangTag) {
	sub := make(problem.Submission)
	preview := make(map[string]string)
	language := -1
	for key, val := range config {
		str, ok := ctx.GetPostForm(key + "_text")
		name := key
		//get language
		if val.Accepted == utils.Csource {
			lang, ok := libs.PostIntRange(ctx, key+"_lang", 0, len(libs.LangSuf)-1)
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
			preview[key] = getPreview(byt, val.Accepted)
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
			preview[key] = getPreview(w.Bytes(), val.Accepted)
			sub.SetSource(workflow.Gsubm, key, name, w)
		}
	}
	return sub, preview, utils.LangTag(language)
}

/*
Get previews of submitted contents
*/
func getPreview(val []byte, typ utils.CtntType) string {
	const preview_length = 256
	switch typ {
	case utils.Cbinary:
		if len(val) * 2 <= preview_length {
			return fmt.Sprintf("%X", val)
		}
		return fmt.Sprintf("%X", val[: preview_length / 2]) + "..."
	case utils.Cplain:
		if len(val) <= preview_length {
			return string(val)
		}
		return string(val[: preview_length]) + "..."
	case utils.Csource:
		return string(val)
	}
	return ""
}

/*
Query single submission, when user is in contests which score_private=true(i.e. cannot see full result),
this function will delete extra information.

Along with CalcMethod and SubmissionConfig
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
	if !PRCanSeeWithoutContent(ctx, ret.ProblemId) {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		pro := internal.PRLoad(ret.ProblemId)
		libs.APIWriteBack(ctx, 200, "", map[string]any{"submission": ret, "calcmethod": pro.DataInfo.CalcMethod, "submconfig": pro.SubmConfig})
	}
}
