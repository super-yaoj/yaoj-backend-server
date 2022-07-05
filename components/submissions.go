package components

import (
	"fmt"
	"os"
	"time"
	"yao/controllers"
	"yao/libs"

	"github.com/gin-gonic/gin"
	"github.com/sshwy/yaoj-core/pkg/problem"
	"github.com/sshwy/yaoj-core/pkg/utils"
)

func SMList(ctx *gin.Context) {
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok { return }
	problem_id := libs.GetIntDefault(ctx, "problem_id", 0)
	contest_id := libs.GetIntDefault(ctx, "contest_id", 0)
	submitter := libs.GetIntDefault(ctx, "submitter", 0)
	user_id := GetUserId(ctx)
	columns := "submission_id, submitter, problem_id, contest_id, status, score, time, memory, language, submit_time"
	
	_, isleft := ctx.GetQuery("left")
	bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	query := libs.If(problem_id == 0, "", fmt.Sprintf(" and problem_id=%d", problem_id)) +
		libs.If(contest_id == 0, "", fmt.Sprintf(" and contest_id=%d", contest_id)) +
		libs.If(submitter == 0, "", fmt.Sprintf(" and submitter=%d", submitter))
	must := "1"
	if !ISAdmin(ctx) {
		perms, _ := controllers.USPermissions(user_id)
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
	var submissions []controllers.Submission
	if isleft {
		libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id<=%d and %s %s order by submission_id desc limit %d", columns, bound, must, query, pagesize))
	} else {
		libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id>=%d and %s %s order by submission_id limit %d", columns, bound, must, query, pagesize))
	}
	isfull := len(submissions) == pagesize
	if isfull { submissions = submissions[: pagesize - 1] }
	if !isleft { libs.Reverse(submissions) }
	controllers.SMGetExtraInfo(submissions)
	libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": submissions, "isfull": isfull })
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
	if !ok { return }
	contest_id := libs.PostIntDefault(ctx, "contest_id", 0)
	in_contest, ok := PRCanSee(ctx, problem_id, contest_id)
	if !ok { return }
	if !in_contest { contest_id = 0 }
	
	pro := controllers.PRLoad(problem_id)
	if pro.Id != problem_id {
		libs.APIWriteBack(ctx, 400, "problem has no data", nil)
		return
	}
	tmp := libs.GetTempDir()
	zip := tmp + "/1.zip"
	os.Mkdir(tmp, os.ModePerm)
	defer os.RemoveAll(tmp)
	file, err := ctx.FormFile("all")
	language := -1
	
	if err == nil {
		var tot_size int64 = 0
		for _, val := range pro.SubmConfig {
			tot_size += int64(val.Length)
		}
		if file.Size > tot_size {
			libs.APIWriteBack(ctx, 400, "file too large", nil)
			return
		}
		ctx.SaveUploadedFile(file, zip)
	} else {
		sub := make(problem.Submission)
		for key, val := range pro.SubmConfig {
			str, ok := ctx.GetPostForm(key + "_text")
			path := tmp + "/" + key
			if val.Accepted == utils.Csource {
				lang, ok := libs.PostIntRange(ctx, key + "_lang", 0, len(libs.LangSuf) - 1)
				if !ok { return }
				language = lang
				path += libs.LangSuf[lang]
			}
			if ok {
				if len(str) > int(val.Length) {
					libs.APIWriteBack(ctx, 400, "file " + key + " too large", nil)
					return
				}
				err = os.WriteFile(path, []byte(str), os.ModePerm)
			} else {
				file, err = ctx.FormFile(key + "_file")
				if err != nil {
					libs.APIInternalError(ctx, err)
					return
				}
				if file.Size > int64(val.Length) {
					libs.APIWriteBack(ctx, 400, "file " + key + " too large", nil)
					return
				}
				err = ctx.SaveUploadedFile(file, path)
			}
			if err != nil {
				libs.APIInternalError(ctx, err)
				return
			}
			sub.Set(key, path)
		}
		sub.DumpFile(zip)
	}
	err = controllers.SMCreate(user_id, problem_id, contest_id, language, zip)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func SMQuery(ctx *gin.Context) {
	sid, ok := libs.GetInt(ctx, "submission_id")
	if !ok { return }
	ret, err := controllers.SMQuery(sid)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !PRCanSeeWithoutContent(ctx, ret.ProblemId) {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": ret })
	}
}