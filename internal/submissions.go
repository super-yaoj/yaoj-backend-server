package internal

import (
	"fmt"
	"sort"
	"sync"
	"time"
	"yao/db"

	jsoniter "github.com/json-iterator/go"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	utils "github.com/super-yaoj/yaoj-utils"
)

// 钩子：数据库修改完成后
func AfterSubmCreate(f func(SubmissionBase)) {
	Listen("AfterSubmCreate", f)
}

// 钩子：数据库修改完成后
func AfterSubmJudge(f func(SubmissionBase)) {
	Listen("AfterSubmJudge", f)
}

// 钩子：数据库修改完成后
func AfterSubmDelete(f func(SubmissionBase)) {
	Listen("AfterSubmDelete", f)
}

// 钩子：数据库修改完成后，重新评测前
func OnSubmRejudge(f func(SubmissionBase)) {
	Listen("OnSubmRejudge", f)
}

type ContentPreview struct {
	Accepted utils.CtntType
	Language utils.LangTag
	Content  string
}

type SubmissionDetails struct {
	ContentPreview string `db:"content_preview" json:"content_preview"` //json encoding of ContentPreview
	Result         string `db:"result" json:"result"`
	PretestResult  string `db:"pretest_result" json:"pretest_result"`
	ExtraResult    string `db:"extra_result" json:"extra_result"`
}

type Submission struct {
	SubmissionBase
	ProblemName   string    `db:"problem_name" json:"problem_name"`
	SubmitterName string    `db:"submitter_name" json:"submitter_name"`
	Rating        int       `db:"rating" json:"rating"`
	Status        int       `db:"status" json:"status"`
	Score         float64   `db:"score" json:"score"`
	SubmitTime    time.Time `db:"submit_time" json:"submit_time"`
	Language      int       `db:"language" json:"language"`
	Time          int       `db:"time" json:"time"`
	Memory        int       `db:"memory" json:"memory"`
	Preview       string    `db:"content_preview" json:"preview"`
	SampleScore   float64   `db:"sample_score" json:"sample_score"`
	Accepted      int       `db:"accepted" json:"accepted"`
	Length        int       `db:"length" json:"length"`

	Details SubmissionDetails `json:"details"`
	Uuid    int64             //useless field for submission query
}

type SubmissionBase struct {
	Id        int `db:"submission_id" json:"submission_id"`
	ProblemId int `db:"problem_id" json:"problem_id"`
	ContestId int `db:"contest_id" json:"contest_id"`
	Submitter int `db:"submitter" json:"submitter"`
}

const (
	PretestAccepted = 1
	TestsAccepted   = 2
	ExtraAccepted   = 4
	Accepted        = 7
	submColumns     = "submission_id, submitter, problem_id, contest_id, status, score, time, memory, language, submit_time, sample_score, accepted, length"
)

/*
Priority: in contest > not in contest, real-time judge > rejudge, pretest > tests > extra

in contest, real-time, pretest judge > custom test > in contest, real-time data judge
*/
func SubmPriority(contest, rejudge bool, mode string) int {
	switch mode {
	case "custom_test":
		return 165
	case "pretest":
		return utils.If(contest, 100, 0) + utils.If(rejudge, 0, 50) + 20
	case "tests":
		return utils.If(contest, 100, 0) + utils.If(rejudge, 0, 50) + 10
	case "extra":
		return utils.If(contest, 100, 0) + utils.If(rejudge, 0, 50)
	}
	return 0
}

/*
Process a whole judge on single problem submission
*/
func SubmJudge(sub SubmissionBase, rejudge bool, uuid int64) error {
	for _, val := range []string{"pretest", "tests", "extra"} {
		InsertSubmission(int(sub.Id), uuid, SubmPriority(sub.ContestId > 0, rejudge, val), val)
	}
	return nil
}

func SubmCreate(user_id, problem_id, contest_id int, language utils.LangTag, zipfile []byte, preview map[string]ContentPreview, length int) error {
	current := utils.TimeStamp()
	id, err := db.InsertGetId("insert into submissions values (null, ?, ?, ?, ?, 0, -1, -1, ?, ?, 0, 0, ?, ?)", user_id, problem_id, contest_id, Waiting, language, time.Now(), current, length)
	if err != nil {
		return err
	}
	js, err := jsoniter.Marshal(preview)
	if err != nil {
		return err
	}
	_, err = db.Exec("insert into submission_details values (?, ?, ?, \"\", \"\", \"\")", id, zipfile, js)
	if err != nil {
		return err
	}
	sb := SubmissionBase{int(id), problem_id, contest_id, user_id}
	Register("AfterSubmCreate", sb)
	return SubmJudge(sb, false, current)
}

func SubmListByIds(subids []int) []Submission {
	if len(subids) == 0 {
		return []Submission{}
	}
	sidmap := make(map[int]int)
	for i, j := range subids {
		sidmap[j] = i
	}
	rows, err := db.Query("select " + submColumns + " from submissions where submission_id in (" + utils.JoinArray(subids) + ")")
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer rows.Close()
	subs := make([]Submission, len(subids))
	for rows.Next() {
		var cur Submission
		rows.StructScan(&cur)
		subs[sidmap[cur.Id]] = cur
	}
	SubmGetExtraInfo(subs)
	return subs
}

/*
Get problem title and user name by id avoiding querying in the database one by one
*/
func SubmGetExtraInfo(subs []Submission) {
	if len(subs) == 0 {
		return
	}
	probs, users := []int{}, []int{}
	for _, val := range subs {
		probs = append(probs, val.ProblemId)
		users = append(users, val.Submitter)
	}
	type Name struct {
		Id     int    `db:"id"`
		Name   string `db:"name"`
		Rating int    `db:"rating"`
	}
	var pname, uname []Name
	db.SelectAll(&pname, "select problem_id as id, title as name from problems where problem_id in ("+utils.JoinArray(probs)+")")
	db.SelectAll(&uname, "select user_id as id, user_name as name, rating from user_info where user_id in ("+utils.JoinArray(users)+")")
	sort.Slice(pname, func(i, j int) bool { return pname[i].Id < pname[j].Id })
	sort.Slice(uname, func(i, j int) bool { return uname[i].Id < uname[j].Id })
	for key, val := range subs {
		pid := sort.Search(len(pname), func(i int) bool { return pname[i].Id >= val.ProblemId })
		uid := sort.Search(len(uname), func(i int) bool { return uname[i].Id >= val.Submitter })
		subs[key].ProblemName, subs[key].SubmitterName, subs[key].Rating = pname[pid].Name, uname[uid].Name, uname[uid].Rating
	}
}

func SubmGetBaseInfo(submission_id int) (SubmissionBase, error) {
	ret := SubmissionBase{Id: submission_id}
	err := db.SelectSingle(&ret, "select problem_id, contest_id, submitter from submissions where submission_id=?", submission_id)
	return ret, err
}

func SubmQuery(sid int) (Submission, error) {
	var ret Submission
	err := db.SelectSingle(&ret, "select * from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return ret, err
	}
	err = db.SelectSingle(&ret.Details, "select content_preview, result, pretest_result, extra_result from submission_details where submission_id=?", sid)
	if err != nil {
		return ret, err
	}
	err = db.SelectSingle(&ret, "select title as problem_name from problems where problem_id=?", ret.ProblemId)
	if err != nil {
		return ret, err
	}
	err = db.SelectSingle(&ret, "select user_name as submitter_name from user_info where user_id=?", ret.Submitter)
	return ret, err
}

/*
For params problem_id, contest_id, submitter, if you do not want to limit them then just leave them as 0.
user_id is the current user's id
*/
func SubmList(bound, pagesize, user_id, submitter, problem_id, contest_id int, isleft, isadmin bool) ([]Submission, bool, error) {
	query := utils.If(problem_id == 0, "", fmt.Sprintf(" and problem_id=%d", problem_id)) +
		utils.If(contest_id == 0, "", fmt.Sprintf(" and contest_id=%d", contest_id)) +
		utils.If(submitter == 0, "", fmt.Sprintf(" and submitter=%d", submitter))
	must := "1"
	if !isadmin {
		perms, err := UserPermissions(user_id)
		if err != nil {
			return nil, false, err
		}
		perm_str := utils.JoinArray(perms)
		//problems user can see
		probs, err := db.SelectInts("select problem_id from problem_permissions where permission_id in (" + perm_str + ")")
		if err != nil {
			return nil, false, err
		}
		/*
			First, user can see all finished contests
			For running contests, participants cannnot see other's contest submissions if score_private=1
			For not started contests, they must contain no contest submissions according to the definition, so we can discard them
		*/
		conts, err := db.SelectInts("select contest_id from contest_permissions where permission_id in (" + perm_str + ")")
		if err != nil {
			return nil, false, err
		}
		//remove contests that cannot see(i.e. the running contests with score_private=true)
		conts_running, err := db.SelectInts("select a.contest_id from ((select contest_id from contests where start_time<=? and end_time>=? and score_private=1) as a join (select contest_id from contest_participants where user_id=?) as b on a.contest_id=b.contest_id)", time.Now(), time.Now(), user_id)
		if err != nil {
			return nil, false, err
		}
		for i := range conts {
			//running contests is few, so brute force is just ok
			if utils.HasElement(conts_running, conts[i]) {
				conts[i] = 0
			}
		}

		must = "("
		if problem_id == 0 {
			must += utils.If(len(probs) == 0, "0", "(problem_id in ("+utils.JoinArray(probs)+"))")
		} else {
			must += utils.If(utils.HasElement(probs, problem_id), "1", "0")
		}
		if contest_id == 0 {
			must += utils.If(len(conts) == 0, " or 0", " or (contest_id in ("+utils.JoinArray(conts)+"))")
		} else {
			must += " or " + utils.If(utils.HasElement(conts, contest_id), "1", "0")
		}
		if submitter == 0 {
			if user_id > 0 {
				must += fmt.Sprintf(" or submitter=%d)", user_id)
			} else {
				must += ")"
			}
		} else {
			must += utils.If(submitter == user_id, " or 1)", ")")
		}
	}
	pagesize += 1
	var submissions []Submission
	var err error
	if isleft {
		err = db.SelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id<=%d and %s %s order by submission_id desc limit %d", submColumns, bound, must, query, pagesize))
	} else {
		err = db.SelectAll(&submissions, fmt.Sprintf("select %s from submissions where submission_id>=%d and %s %s order by submission_id limit %d", submColumns, bound, must, query, pagesize))
	}
	if err != nil {
		return nil, false, err
	}
	isfull := len(submissions) == pagesize
	if isfull {
		submissions = submissions[:pagesize-1]
	}
	if !isleft {
		utils.Reverse(submissions)
	}
	SubmGetExtraInfo(submissions)
	return submissions, isfull, nil
}

func SubmPretestOnly(sub *Submission) {
	sub.Score = sub.SampleScore
	sub.Details.ExtraResult, sub.Details.Result = "", ""
	sub.Time, sub.Memory = -1, -1
	if (sub.Status & JudgingPretest) != 0 {
		sub.Status = Finished
	}
}

var sm_update_mutex = sync.Mutex{}

func SubmUpdate(sid, pid int, mode string, result []byte) error {
	sm_update_mutex.Lock()
	defer sm_update_mutex.Unlock()
	prob := ProbLoad(pid)
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
	accepted := true
	var err error
	res_map := make(map[string]any)
	err = jsoniter.Unmarshal(result, &res_map)
	is_subtask, has_data := res_map["IsSubtask"].(bool)

	if err == nil && has_data {
		for _, subtask := range res_map["Subtask"].([]any) {
			var sub_score float64
			first := true
			for _, test := range subtask.(map[string]any)["Testcase"].([]any) {
				test_score := test.(map[string]any)["Score"].(float64)
				time_used += test.(map[string]any)["Time"].(float64)
				memory_used = utils.Max(memory_used, test.(map[string]any)["Memory"].(float64))
				if first {
					sub_score = test_score
					first = false
					continue
				}
				if is_subtask {
					switch testdata.CalcMethod {
					case problem.Mmin:
						sub_score = utils.Min(sub_score, test_score)
					case problem.Mmax:
						sub_score = utils.Max(sub_score, test_score)
					case problem.Msum:
						sub_score += test_score
					}
				} else {
					sub_score += test_score
				}
			}
			score += sub_score
			if sub_score != subtask.(map[string]any)["Fullscore"].(float64) {
				accepted = false
			}
		}
	}

	//'and status>=0' means when meets an internal error, we shouldn't update status
	if mode == "tests" {
		_, err = db.Exec("update submissions set status=status|?, accepted=accepted|?, score=?, time=?, memory=? where submission_id=? and status>=0",
			JudgingTests, utils.If(accepted, TestsAccepted, 0), score, int(time_used/float64(time.Millisecond)), int(memory_used/1024), sid)
	} else if mode == "pretest" {
		_, err = db.Exec("update submissions set status=status|?, accepted=accepted|?, sample_score=? where submission_id=? and status>=0",
			JudgingPretest, utils.If(accepted, PretestAccepted, 0), score, sid)
	} else {
		_, err = db.Exec("update submissions set status=status|?, accepted=accepted|? where submission_id=? and status>=0",
			JudgingExtra, utils.If(accepted, ExtraAccepted, 0), sid)
	}
	if err != nil {
		return err
	}
	_, err = db.Exec("update submission_details set "+column_name+"=? where submission_id=?", result, sid)
	if err != nil {
		return err
	}
	var subinfo struct {
		SubmissionBase
		Status int `db:"status"`
	}
	err = db.SelectSingle(&subinfo, "select submission_id, problem_id, contest_id, status from submissions where submission_id=?", sid)
	if err != nil {
		return err
	}
	//we should ensure that each submission will be exactly updated once
	if subinfo.Status == Finished || (subinfo.Status < 0 && mode == "tests") {
		Register("AfterSubmJudge", subinfo.SubmissionBase)
	}
	return nil
}

func SubmJudgeCustomTest(content []byte) []byte {
	callback := make(chan []byte)
	//find a free submission_id
	sid, err := db.InsertGetId("insert into custom_tests values (null, ?)", content)
	if err != nil {
		fmt.Println(err)
		return []byte{}
	}
	InsertCustomTest(int(sid), &callback)
	result := <-callback
	go db.Exec("delete from custom_tests where id=?", sid)
	return result
}

func SubmDelete(sub SubmissionBase) error {
	_, err := db.Exec("delete from submissions where submission_id=?", sub.Id)
	if err != nil {
		return err
	}
	_, err = db.Exec("delete from submission_details where submission_id=?", sub.Id)
	if err != nil {
		return err
	}
	Register("AfterSubmDelete", sub)
	return nil
}

func SubmRejudge(submission_id int) error {
	sub, err := SubmGetBaseInfo(submission_id)
	if err != nil {
		return err
	}
	//update uuid to cancel other entries in the judging queue
	current := utils.TimeStamp()
	_, err = db.Exec("update submissions set uuid=?, status=0, accepted=0 where submission_id=?", current, submission_id)
	if err != nil {
		return err
	}
	Register("OnSubmRejudge", sub)
	return SubmJudge(sub, true, current)
}

func SubmRemoveTestDetails(js string) string {
	if js == "" {
		return ""
	}
	var val map[string]any
	jsoniter.UnmarshalFromString(js, &val)
	if val["Subtask"] == nil {
		return ""
	}
	for _, subtask := range val["Subtask"].([]any) {
		if subtask.(map[string]any)["Testcase"] == nil {
			continue
		}
		for _, test := range subtask.(map[string]any)["Testcase"].([]any) {
			test.(map[string]any)["File"] = nil
		}
	}
	ret, err := jsoniter.MarshalToString(val)
	if err != nil {
		fmt.Println(err)
	}
	return ret
}

func SubmExists(subm_id int) bool {
	cnt, _ := db.SelectSingleInt("select count(*) from submissions where submission_id=?", subm_id)
	return cnt > 0
}
