package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
	"yao/libs"

	"github.com/super-yaoj/yaoj-core/pkg/problem"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
)

type ContentPreview struct {
	Accepted 	utils.CtntType
	Language 	utils.LangTag
	Content 	string
}

type SubmissionDetails struct {
	ContentPreview string `db:"content_preview" json:"content_preview"`
	Result 		   string `db:"result" json:"result"`
	PretestResult  string `db:"pretest_result" json:"pretest_result"`
	ExtraResult    string `db:"extra_result" json:"extra_result"`
}

type Submission struct {
	Id            int       `db:"submission_id" json:"submission_id"`
	Submitter     int       `db:"submitter" json:"submitter"`
	ProblemId     int       `db:"problem_id" json:"problem_id"`
	ProblemName   string    `db:"problem_name" json:"problem_name"`
	ContestId     int       `db:"contest_id" json:"contest_id"`
	SubmitterName string    `db:"submitter_name" json:"submitter_name"`
	Status        int       `db:"status" json:"status"`
	Score         float64   `db:"score" json:"score"`
	SubmitTime    time.Time `db:"submit_time" json:"submit_time"`
	Language      int       `db:"language" json:"language"`
	Time          int       `db:"time" json:"time"`
	Memory        int       `db:"memory" json:"memory"`
	Preview 	  string	`db:"content_preview" json:"preview"`
	SampleScore   float64 	`db:"sample_score" json:"sample_score"`
	Hacked 		  bool 		`db:"hacked" json:"hacked"`
	
	Details 	  SubmissionDetails `json:"details"`
}

func SMGetZipName(submission_id int) string {
	return libs.TmpDir + fmt.Sprintf("submission_%d.zip", submission_id)
}

//Priority: in contest > not in contest, pretest > data > extra
func JudgeSubmission(submission_id, problem_id, contest_id int) error {
	pro := PRLoad(problem_id)
	status := 0
	if PRHasPretest(pro) {
		InsertSubmission(int(submission_id), libs.If(contest_id > 0, 3, 0) + 3, "pretest")
	} else {
		status |= JudgingPretest
	}
	if PRHasData(pro) {
		InsertSubmission(int(submission_id), libs.If(contest_id > 0, 3, 0) + 2, "tests")
	} else {
		return errors.New("Problem has no data!")
	}
	if PRHasExtra(pro) {
		InsertSubmission(int(submission_id), libs.If(contest_id > 0, 3, 0) + 1, "extra")
	} else {
		status |= JudgingExtra
	}
	if status != 0 {
		_, err := libs.DBUpdate("update submissions set status=status|? where submission_id=?", status, submission_id)
		if err != nil {
			return err
		}
	}
	return nil
}

func SMCreate(user_id, problem_id, contest_id int, language utils.LangTag, zipfile []byte, preview map[string]ContentPreview) error {
	id, err := libs.DBInsertGetId("insert into submissions values (null, ?, ?, ?, ?, 0, -1, -1, ?, ?, 0, 0)", user_id, problem_id, contest_id, Waiting, language, time.Now())
	if err != nil {
		return err
	}
	js, err := json.Marshal(preview)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("insert into submission_details values (?, ?, ?, \"\", \"\", \"\")", id, zipfile, js)
	if err != nil {
		return err
	}
	return JudgeSubmission(int(id), problem_id, contest_id)
}

/*
Get problem title and user name by id avoiding querying in the database one by one
*/
func SMGetExtraInfo(subs []Submission) {
	if len(subs) == 0 {
		return
	}
	probs, users := []int{}, []int{}
	for _, val := range subs {
		probs = append(probs, val.ProblemId)
		users = append(users, val.Submitter)
	}
	type Name struct {
		Id   int    `db:"id"`
		Name string `db:"name"`
	}
	var pname, uname []Name
	libs.DBSelectAll(&pname, "select problem_id as id, title as name from problems where problem_id in (" + libs.JoinArray(probs) + ")")
	libs.DBSelectAll(&uname, "select user_id as id, user_name as name from user_info where user_id in (" + libs.JoinArray(users) + ")")
	sort.Slice(pname, func(i, j int) bool { return pname[i].Id < pname[j].Id })
	sort.Slice(uname, func(i, j int) bool { return uname[i].Id < uname[j].Id })
	for key, val := range subs {
		pid := sort.Search(len(pname), func(i int) bool { return pname[i].Id >= val.ProblemId })
		uid := sort.Search(len(uname), func(i int) bool { return uname[i].Id >= val.Submitter })
		subs[key].ProblemName, subs[key].SubmitterName = pname[pid].Name, uname[uid].Name
	}
}

func SMQuery(sid int) (Submission, error) {
	var ret Submission
	err := libs.DBSelectSingle(&ret, "select * from submissions where submission_id=?", sid)
	if err != nil {
		return ret, err
	}
	err = libs.DBSelectSingle(&ret.Details, "select content_preview, result, pretest_result, extra_result from submission_details where submission_id=?", sid)
	if err != nil {
		return ret, err
	}
	err = libs.DBSelectSingle(&ret, "select title as problem_name from problems where problem_id=?", ret.ProblemId)
	if err != nil {
		return ret, err
	}
	err = libs.DBSelectSingle(&ret, "select user_name as submitter_name from user_info where user_id=?", ret.Submitter)
	return ret, err
}

/*
For params problem_id, contest_id, submitter, if you do not want to limit them then just leave them as 0.
user_id is the current user's id
*/
func SMList(bound, pagesize, user_id, submitter, problem_id, contest_id int, isleft, isadmin bool) ([]Submission, bool, error) {
	const columns = "submission_id, submitter, problem_id, contest_id, status, score, time, memory, language, submit_time, sample_score, hacked"
	
	query := libs.If(problem_id == 0, "", fmt.Sprintf(" and problem_id=%d", problem_id)) +
		libs.If(contest_id == 0, "", fmt.Sprintf(" and contest_id=%d", contest_id)) +
		libs.If(submitter == 0, "", fmt.Sprintf(" and submitter=%d", submitter))
	must := "1"
	if !isadmin {
		perms, err := USPermissions(user_id)
		if err != nil {
			return nil, false, err
		}
		perm_str := libs.JoinArray(perms)
		//problems user can see
		probs, err := libs.DBSelectInts("select problem_id from problem_permissions where permission_id in (" + perm_str + ")")
		if err != nil {
			return nil, false, err
		}
		/*
			First, user can see all finished contests
			For running contests, participants cannnot see other's contest submissions if score_private=1
			For not started contests, they must contain no contest submissions according to the definition, so we can discard them
		*/
		conts, err := libs.DBSelectInts("select contest_id from contest_permissions where permission_id in (" + perm_str + ")")
		if err != nil {
			return nil, false, err
		}
		//remove contests that cannot see(i.e. the running contests with score_private=true)
		conts_running, err := libs.DBSelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and score_private=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), user_id)
		if err != nil {
			return nil, false, err
		}
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
			must += libs.If(len(conts) == 0, " or 0", " or (contest_id in (" + libs.JoinArray(conts) + "))")
		} else {
			must += " or " + libs.If(libs.HasIntN(conts, contest_id), "1", "0")
		}
		if submitter == 0 {
			if user_id > 0 {
				must += fmt.Sprintf(" or submitter=%d)", user_id)
			} else {
				must += ")"
			}
		} else {
			must += libs.If(submitter == user_id, " or 1)", ")")
		}
	}
	pagesize += 1
	var submissions []Submission
	var err error
	if isleft {
		err = libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id<=%d and %s %s order by submission_id desc limit %d", columns, bound, must, query, pagesize))
	} else {
		err = libs.DBSelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id>=%d and %s %s order by submission_id limit %d", columns, bound, must, query, pagesize))
	}
	if err != nil {
		return nil, false, err
	}
	isfull := len(submissions) == pagesize
	if isfull {
		submissions = submissions[:pagesize-1]
	}
	if !isleft {
		libs.Reverse(submissions)
	}
	SMGetExtraInfo(submissions)
	return submissions, isfull, nil
}

func SMPretestOnly(sub *Submission) {
	sub.Score = sub.SampleScore
	sub.Details.ExtraResult, sub.Details.Result = "", ""
	sub.Time, sub.Memory = -1, -1
	if (sub.Status & JudgingPretest) != 0 {
		sub.Status = Finished
	}
}

func SMUpdate(sid, pid int, mode string, result []byte) error {
	res_map := make(map[string]any)
	err := json.Unmarshal(result, &res_map)
	if err != nil {
		return err
	}
	prob := PRLoad(pid)
	var testdata *problem.TestdataInfo
	var column_name string
	switch mode {
	case "pretest":
		testdata = &prob.DataInfo.Pretest
		column_name = "pretest_result"
	case "tests":
		testdata = &prob.DataInfo.TestdataInfo
		column_name = "result"
	case "extra":
		testdata = &prob.DataInfo.Extra
		column_name = "extra_result"
	}

	var score, time_used, memory_used float64 = 0, 0, 0
	is_subtask := res_map["IsSubtask"].(bool)
	for _, subtask := range res_map["Subtask"].([]any) {
		var sub_score float64
		first := true
		for _, test := range subtask.(map[string]any)["Testcase"].([]any) {
			test_score := test.(map[string]any)["Score"].(float64)
			time_used += test.(map[string]any)["Time"].(float64)
			memory_used = math.Max(memory_used, test.(map[string]any)["Memory"].(float64))
			if first {
				sub_score = test_score
				first = false
				continue
			}
			if is_subtask {
				switch testdata.CalcMethod {
				case problem.Mmin:
					sub_score = math.Min(sub_score, test_score)
				case problem.Mmax:
					sub_score = math.Max(sub_score, test_score)
				case problem.Msum:
					sub_score += test_score
				}
			} else {
				sub_score += test_score
			}
		}
		score += sub_score
	}
	
	if mode == "tests" {
		_, err = libs.DBUpdate("update submissions set status=status|?, score=?, time=?, memory=? where submission_id=?",
			JudgingTests, score, int(time_used/float64(time.Millisecond)), int(memory_used/1024), sid)
	} else if mode == "pretest" {
		_, err = libs.DBUpdate("update submissions set status=status|?, sample_score=? where submission_id=?", JudgingPretest, score, sid)
	} else {
		_, err = libs.DBUpdate("update submissions set status=status|? where submission_id=?", JudgingExtra, sid)
	}
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("update submission_details set " + column_name + "=? where submission_id=?", result, sid)
	return err
}