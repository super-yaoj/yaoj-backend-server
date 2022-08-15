package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"yao/config"
	"yao/db"

	jsoniter "github.com/json-iterator/go"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
	"github.com/super-yaoj/yaoj-utils/pq"
)

type Judger struct {
	url string
	//unique random judger id, change after each request
	jid      string
	callback chan []byte
}

type JudgeEntry struct {
	sid      int
	mode     string       //one of "pretest", "tests", "extra", "custom_test"
	uuid     int64        //if mode != "custom_test"(i.e. normal submission), uuid means whether this submission is the recent entry in the judging queue(set by time-stamp)
	callback *chan []byte //if mode="custom_test", you should give a callback channel which returns the result
}

func NewJudger(url string) *Judger {
	return &Judger{url, utils.RandomString(64), make(chan []byte)}
}

var judgers = []*Judger{
	NewJudger("http://localhost:3000"),
	// "http://localhost:8083",
	// "http://localhost:8084",
}

var waitingList = pq.NewBlockPriorityQueue[*JudgeEntry]()

// 1 on each bit means that the corresponding status has finished
const (
	InternalError  = -1
	Waiting        = 1
	JudgingPretest = 2
	JudgingTests   = 4
	JudgingExtra   = 8
	Finished       = 15
)

func JudgersInit() {
	var sub []Submission
	err := db.SelectAll(&sub, "select submission_id, problem_id, contest_id, uuid from submissions where status < ? and status >= 0", Finished)
	if err != nil {
		log.Fatal(err)
	}
	for _, val := range sub {
		err := SubmJudge(val.SubmissionBase, true, val.Uuid)
		if err != nil {
			fmt.Println(err)
		}
	}
	for _, i := range judgers {
		go judgerStart(i)
	}
}

type judgerResponse struct {
	Err      string `json:"error"`
	Err_code int    `json:"error_code"`
	Msg      string `json:"message"`
}

func judgerStart(judger *Judger) {
	for {
		//wait for a submission
		subm := waitingList.Pop()
		sid, uuid, mode := subm.sid, subm.uuid, subm.mode
		if mode != "custom_test" {
			if !judgeSubmission(sid, uuid, mode, judger) {
				db.Exec("update submissions set status=? where submission_id=?", InternalError, sid)
				db.Exec("update submission_details set result=\"\", pretest_result=\"\", extra_result=\"\" where submission_id=?", sid)
				sub, _ := SubmGetBaseInfo(sid)
				SubmUpdate(sid, sub.ProblemId, subm.mode, []byte{})
			}
		} else {
			judgeCustomTest(sid, subm.callback, judger)
		}
		//change the judger id to avoid attacks
		judger.jid = utils.RandomString(64)
	}
}

// return true or false indicates whether judging succeeds
func judgeSubmission(sid int, uuid int64, mode string, judger *Judger) bool {
	type TempInfo struct {
		Prob int   `db:"problem_id"`
		Uuid int64 `db:"uuid"`
	}
	var tinfo TempInfo
	err := db.SelectSingle(&tinfo, "select problem_id, uuid from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if tinfo.Uuid != uuid {
		//this judge entry isn't the recent entry in the judging queue
		return true
	}

	pro := ProbLoad(tinfo.Prob)
	if !ProbHasData(pro, mode) {
		go SubmUpdate(sid, tinfo.Prob, mode, []byte{})
		return true
	}
	var content []byte
	err = db.SelectSingleColumn(&content, "select content from submission_details where submission_id=?", sid)
	go db.Exec("update submissions set status=status|? where submission_id=?", Waiting, sid)
	var check_sum string
	err1 := db.SelectSingleColumn(&check_sum, "select check_sum from problems where problem_id=?", tinfo.Prob)
	if err != nil || err1 != nil {
		fmt.Println(err, err1)
		return false
	}

	for { //Repeating for data sync
		//get a new judger id
		judger.jid = utils.RandomString(64)
		res, err := http.Post(judger.url+"/judge?"+getQuery(map[string]string{
			"mode": mode,
			"sum":  check_sum,
			"cb":   fmt.Sprintf(config.Global.BackDomain+"/FinishJudging?jid=%s", judger.jid),
		}), "binary", bytes.NewBuffer(content))
		if err != nil {
			fmt.Printf("%v\n", err)
			return false
		}
		body, _ := ioutil.ReadAll(res.Body)
		var jr judgerResponse
		jsoniter.Unmarshal(body, &jr)

		if jr.Msg == "ok" {
			break
		} else if jr.Err_code == 1 {
			ProblemRWLock.RLock(tinfo.Prob)
			file, err := os.Open(ProbGetDataZip(tinfo.Prob))
			res, err1 = http.Post(judger.url+"/sync?"+getQuery(map[string]string{"sum": check_sum}), "binary", file)
			ProblemRWLock.RUnlock(tinfo.Prob)
			if err != nil || err1 != nil {
				fmt.Printf("%v %v\n", err, err1)
				return false
			}
			body, _ = ioutil.ReadAll(res.Body)
			jsoniter.Unmarshal(body, &jr)
			if jr.Msg != "ok" {
				fmt.Printf("%s\n", jr.Err)
				return false
			}
		} else {
			fmt.Printf("%s\n", jr.Err)
			return false
		}
	}
	//Waiting judger finishes
	ret := <-judger.callback
	err = db.SelectSingleColumn(&tinfo.Uuid, "select uuid from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if tinfo.Uuid == uuid {
		//Update status if and only if this is the recent submission
		go func() {
			err := SubmUpdate(sid, tinfo.Prob, mode, ret)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}()
	}
	return true
}

func judgeCustomTest(sid int, callback *chan []byte, judger *Judger) {
	var content []byte
	err := db.SelectSingleColumn(&content, "select content from custom_tests where id=?", sid)
	if err != nil {
		fmt.Println(err)
		*callback <- []byte{}
		return
	}
	res, err := http.Post(judger.url+"/custom?"+getQuery(map[string]string{
		"cb": fmt.Sprintf(config.Global.BackDomain+"/FinishJudging?jid=%s", judger.jid),
	}), "binary", bytes.NewBuffer(content))
	if err != nil {
		fmt.Println(err)
		*callback <- []byte{}
		return
	}
	body, _ := ioutil.ReadAll(res.Body)
	var jr judgerResponse
	jsoniter.Unmarshal(body, &jr)
	if jr.Msg != "ok" {
		fmt.Println(jr.Err)
		*callback <- []byte{}
		return
	}
	*callback <- <-judger.callback
}

func InsertSubmission(sid int, uuid int64, priority int, mode string) {
	waitingList.Push(&JudgeEntry{sid, mode, uuid, nil}, priority)
}

func InsertCustomTest(sid int, callback *chan []byte) {
	waitingList.Push(&JudgeEntry{sid, "custom_test", 0, callback}, 0)
}

func FinishJudging(jid string, result []byte) error {
	for i := 0; i < 5; i++ {
		for key := range judgers {
			if judgers[key].jid == jid {
				judgers[key].callback <- result
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("No such judger: judger_id=%s", jid)
}

func JudgerLog(id int) string {
	if id >= len(judgers) {
		return "No such judger"
	}
	res, err := http.Get(judgers[id].url + "/log")
	if err != nil {
		return "Internal server error"
	}
	body, _ := ioutil.ReadAll(res.Body)
	return string(body)
}

func getQuery(query map[string]string) string {
	first := true
	ret := strings.Builder{}
	for key, val := range query {
		if !first {
			ret.WriteString("&")
		} else {
			first = false
		}
		ret.WriteString(key + "=" + url.QueryEscape(val))
	}
	return ret.String()
}
