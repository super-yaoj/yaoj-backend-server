package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
	"yao/internal"
	"yao/libs"

	"github.com/goccy/go-json"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
	"github.com/super-yaoj/yaoj-core/pkg/workflow"
)

type SubmListParam struct {
	Auth
	Page 		 `validate:"pagecanbound"`
	ProbID   int `query:"problem_id"`
	CtstID   int `query:"contest_id"`
	Submtter int `query:"submitter"`
}

func SubmList(ctx Context, param SubmListParam) {
	submissions, isfull, err := internal.SMList(
		param.Bound(), *param.PageSize, param.UserID, param.Submtter, param.ProbID, param.CtstID,
		param.IsLeft(), param.IsAdmin(),
	)
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
	//Modify scores to sample_scores when users are in pretest-only contests
	contest_pretest, _ := libs.DBSelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and pretest=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), param.UserID)
	for key := range submissions {
		if libs.HasElement(contest_pretest, submissions[key].ContestId) {
			internal.SMPretestOnly(&submissions[key])
		}
	}
	ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": submissions, "isfull": isfull})
}

type SubmAddParam struct {
	Auth
	ProbID  int     `body:"problem_id" validate:"required"`
	CtstID  int     `body:"contest_id"`
	SubmAll *string `body:"submit_all"`
}

// users can submit from a contest if and only if the contest is running and
// he takes part in the contest (only these submissions are contest submissions)
func SubmAdd(ctx Context, param SubmAddParam) {
	param.NewPermit().AsNormalUser().TrySeeProb(param.ProbID, param.CtstID).Success(func(a any) {
		ctstid := a.(int)
		pro := internal.PRLoad(param.ProbID)
		if !internal.PRHasData(pro, "tests") {
			ctx.JSONAPI(http.StatusBadRequest, "problem has no data", nil)
			return
		}
		var sub problem.Submission
		var language utils.LangTag
		var preview map[string]internal.ContentPreview
		var length int
		if param.SubmAll != nil {
			sub, preview, language, length = parseZipFile(ctx, "all.zip", pro.SubmConfig)
		} else {
			sub, preview, language, length = parseMultiFiles(ctx, pro.SubmConfig)
		}
		if sub == nil {
			return
		}

		w := bytes.NewBuffer(nil)
		sub.DumpTo(w)
		err := internal.SMCreate(param.UserID, param.ProbID, ctstid, language, w.Bytes(), preview, length)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

// When the submitted file is a zip file
func parseZipFile(ctx Context, field string, config internal.SubmConfig) (problem.Submission, map[string]internal.ContentPreview, utils.LangTag, int) {
	file, header, err := ctx.Request.FormFile(field)
	if err != nil {
		return nil, nil, 0, 0
	}
	language := -1
	sub := make(problem.Submission)
	preview := make(map[string]internal.ContentPreview)

	var tot_size int64 = 0
	for _, val := range config {
		tot_size += int64(val.Length)
	}
	if header.Size > tot_size {
		ctx.JSONAPI(http.StatusBadRequest, "file too large", nil)
		return nil, nil, 0, 0
	}

	w := bytes.NewBuffer(nil)
	_, err = io.Copy(w, file)
	if err != nil {
		ctx.ErrorAPI(err)
		return nil, nil, 0, 0
	}
	ret, err := libs.UnzipMemory(w.Bytes())
	if err != nil {
		ctx.JSONAPI(http.StatusBadRequest, "invalid zip file: "+err.Error(), nil)
		return nil, nil, 0, 0
	}
	length := 0
	for name, val := range ret {
		matched := ""
		length += len(val)
		//find corresponding key
		for key := range config {
			if key == name || libs.StartsWith(name, key+".") {
				matched = key
				break
			}
		}
		if matched == "" {
			ctx.JSONAPI(http.StatusBadRequest, "no field matches file name: "+name, nil)
			return nil, nil, 0, 0
		}
		//TODO: get language by file suffix
		preview[matched] = getPreview(val, config[matched].Accepted, -1)
		sub.SetSource(workflow.Gsubm, matched, name, bytes.NewReader(val))
	}
	return sub, preview, utils.LangTag(language), length
}

/*
When user submits files one by one
*/
func parseMultiFiles(ctx Context, config internal.SubmConfig) (problem.Submission, map[string]internal.ContentPreview, utils.LangTag, int) {
	sub := make(problem.Submission)
	preview := make(map[string]internal.ContentPreview)
	language := -1
	length := 0
	for key, val := range config {
		str, ok := ctx.GetPostForm(key + "_text")
		name, lang := key, -1
		//get language
		if val.Accepted == utils.Csource {
			lang, ok = libs.PostIntRange(ctx.Context, key+"_lang", 0, len(libs.LangSuf)-1)
			if !ok || (val.Langs != nil && !libs.HasElement(val.Langs, utils.LangTag(lang))) {
				return nil, nil, 0, 0
			}
			language = lang
			name += libs.LangSuf[lang]
		}
		if ok {
			//text
			length += len(str)
			if len(str) > int(val.Length) {
				ctx.JSONAPI(http.StatusBadRequest, "file "+key+" too large", nil)
				return nil, nil, 0, 0
			}
			byt := []byte(str)
			preview[key] = getPreview(byt, val.Accepted, utils.LangTag(lang))
			sub.SetSource(workflow.Gsubm, key, name, bytes.NewReader(byt))
		} else {
			//file
			file, header, err := ctx.Request.FormFile(key + "_file")
			if err != nil {
				ctx.ErrorAPI(err)
				return nil, nil, 0, 0
			}
			length += int(header.Size)
			if header.Size > int64(val.Length) {
				ctx.JSONAPI(http.StatusBadRequest, "file "+key+" too large", nil)
				return nil, nil, 0, 0
			}
			w := bytes.NewBuffer(nil)
			_, err = io.Copy(w, file)
			if err != nil {
				ctx.ErrorAPI(err)
				return nil, nil, 0, 0
			}
			preview[key] = getPreview(w.Bytes(), val.Accepted, utils.LangTag(lang))
			sub.SetSource(workflow.Gsubm, key, name, w)
		}
	}
	return sub, preview, utils.LangTag(language), length
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
	Auth
	SubmID int `query:"submission_id" validate:"required,submid"`
}

// Query single submission, when user is in contests which score_private=true
// (i.e. cannot see full result), this function will delete extra information.
func SubmGet(ctx Context, param SubmGetParam) {
	param.NewPermit().TrySeeSubm(param.SubmID).Success(func(a any) {
		psubm := a.(PermitSubm)
		//user cannot see submission details inside contests
		if !psubm.CanEdit && !psubm.ByProb {
			if internal.CTPretestOnly(psubm.ContestId) {
				internal.SMPretestOnly(&psubm.Submission)
			} else {
				psubm.Details.Result = internal.SMRemoveTestDetails(psubm.Details.Result)
				psubm.Details.ExtraResult = internal.SMRemoveTestDetails(psubm.Details.ExtraResult)
			}
		}
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"submission": psubm.Submission, "can_edit": psubm.CanEdit})
	}).FailAPIStatusForbidden(ctx)
}

type SubmCustomParam struct {
	Auth
}

func SubmCustom(ctx Context, param SubmCustomParam) {
	param.NewPermit().AsNormalUser().Success(func(any) {
		config := internal.SubmConfig{
			"source": {Langs: nil, Accepted: utils.Csource, Length: 64 * 1024},
			"input":  {Langs: nil, Accepted: utils.Cplain, Length: 10 * 1024 * 1024},
		}
		subm, _, _, _ := parseMultiFiles(ctx, config)
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
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"result": string(result)})
	}).FailAPIStatusForbidden(ctx)
}

type SubmDelParam struct {
	Auth
	SubmID int `query:"submission_id" validate:"required,submid"`
}

func SubmDel(ctx Context, param SubmDelParam) {
	param.NewPermit().TryEditSubm(param.SubmID).Success(func(a any) {
		sub := a.(internal.SubmissionBase)
		err := internal.SMDelete(sub)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type RejudgeParam struct {
	ProbID *int `body:"problem_id" validate:"probid"`
	SubmID *int  `body:"submission_id" validate:"submid"`
	Auth
}

func Rejudge(ctx Context, param RejudgeParam) {
	if param.ProbID != nil {
		param.NewPermit().TryEditProb(*param.ProbID).Success(func(a any) {
			err := internal.PRRejudge(*param.ProbID)
			if err != nil {
				ctx.ErrorRPC(err)
			}
		}).FailRPCStatusForbidden(ctx)
	} else {
		param.NewPermit().TryEditSubm(*param.SubmID).Success(func(any) {
			err := internal.SMRejudge(*param.SubmID)
			if err != nil {
				ctx.ErrorRPC(err)
			}
		})
	}
}
