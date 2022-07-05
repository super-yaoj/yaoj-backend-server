package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
	"yao/libs"

	"github.com/sshwy/yaoj-core/pkg/problem"
)

type Submission struct {
	Id            int       `db:"submission_id" json:"submission_id"`
	Submitter     int       `db:"submitter" json:"submitter"`
	ProblemId     int       `db:"problem_id" json:"problem_id"`
	ProblemName   string    `json:"problem_name"`
	ContestId     int       `db:"contest_id" json:"contest_id"`
	SubmitterName string    `json:"submitter_name"`
	Status        int       `db:"status" json:"status"`
	Score         float64   `db:"score" json:"score"`
	SubmitTime    time.Time `db:"submit_time" json:"submit_time"`
	Result        string    `db:"result" json:"result"`
	Language      int       `db:"language" json:"language"`
	Time          int       `db:"time" json:"time"`
	Memory        int       `db:"memory" json:"memory"`
}

func SMGetZipName(submission_id int) string {
	return libs.TmpDir + fmt.Sprintf("submission_%d.zip", submission_id)
}

func SMCreate(user_id, problem_id, contest_id, language int, tmp_file string) error {
	id, err := libs.DBInsertGetId("insert into submissions values (null, ?, ?, ?, ?, 0, -1, -1, ?, ?, \"\")", user_id, problem_id, contest_id, Waiting, language, time.Now())
	if err != nil {
		return err
	}
	byt, err := os.ReadFile(tmp_file)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("insert into submission_content values (?, ?)", id, byt)
	if err != nil {
		return err
	}
	InsertSubmission(int(id), libs.If(contest_id > 0, 4, 2))
	return nil
}

func SMGetExtraInfo(subs []Submission) {
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
	libs.DBSelectAll(&pname, "select problem_id as id, title as name from problems where problem_id in ("+libs.JoinArray(probs)+")")
	libs.DBSelectAll(&uname, "select user_id as id, user_name as name from user_info where user_id in ("+libs.JoinArray(users)+")")
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
	return ret, err
}

func SMUpdate(sid, pid int, result []byte) error {
	res_map := make(map[string]any)
	err := json.Unmarshal(result, &res_map)
	if err != nil {
		return err
	}
	prob := PRLoad(pid)

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
				switch prob.DataInfo.CalcMethod {
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
	_, err = libs.DBUpdate("update submissions set status=0, score=?, time=?, memory=?, result=? where submission_id=?",
		score, int(time_used/float64(time.Millisecond)), int(memory_used/1024), result, sid)
	return err
}
