package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"yao/libs"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

type Judger struct {
	url 		string
	//unique random judger id, change after each request
	jid 		string
	callback   	chan []byte
}

type JudgeEntry struct {
	sid 		int
	mode 		string //one of "pretest", "tests", "extra", "custom_test"
	uuid 		int64 //if mode != "custom_test"(i.e. normal submission), uuid means whether this submission is the recent entry in the judging queue(set by time-stamp)
	callback 	*chan []byte //if mode="custom_test", you should give a callback channel which returns the result
}

func NewJudger(url string) *Judger {
	return &Judger{url, libs.RandomString(64), make(chan []byte)}
}

var judgers = []*Judger{
	NewJudger("http://localhost:3000"),
	// "http://localhost:8083",
	// "http://localhost:8084",
}

var waitingList = libs.NewBlockPriorityQueue()

//1 on each bit means that the corresponding status has finished
const (
	InternalError 	= -1
	Waiting 		= 1
	JudgingPretest 	= 2
	JudgingTests	= 4
	JudgingExtra 	= 8
	Finished 		= 15
)

func JudgersInit() {
	var sub []Submission
	err := libs.DBSelectAll(&sub, "select submission_id, problem_id, contest_id, uuid from submissions where status < ? and status >= 0", Finished)
	if err != nil {
		log.Fatal(err)
	}
	for _, val := range sub {
		err := SMJudge(val.SubmissionBase, true, val.Uuid)
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
		subm := waitingList.Pop().(JudgeEntry)
		sid, uuid, mode := subm.sid, subm.uuid, subm.mode
		if mode != "custom_test" {
			if !judgeSubmission(sid, uuid, mode, judger) {
				libs.DBUpdate("update submissions set status=? where submission_id=?", InternalError, sid)
				libs.DBUpdate("update submission_details set result=\"\", pretest_result=\"\", extra_result=\"\" where submission_id=?", sid)
			}
		} else {
			judgeCustomTest(sid, subm.callback, judger)
		}
		//change the judger id to avoid attacks
		judger.jid = libs.RandomString(64)
	}
}

//return true or false indicates whether judging succeeds
func judgeSubmission(sid int, uuid int64, mode string, judger *Judger) bool {
	type TempInfo struct {
		Prob int   `db:"problem_id"`
		Uuid int64 `db:"uuid"`
	}
	var tinfo TempInfo
	err := libs.DBSelectSingle(&tinfo, "select problem_id, uuid from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if tinfo.Uuid != uuid {
		//this judge entry isn't the recent entry in the judging queue
		return true
	}
	
	pro := PRLoad(tinfo.Prob)
	if !PRHasData(pro, mode) {
		go SMUpdate(sid, tinfo.Prob, mode, nil)
		return true
	}
	var content []byte
	err = libs.DBSelectSingleColumn(&content, "select content from submission_details where submission_id=?", sid)
	go libs.DBUpdate("update submissions set status=status|? where submission_id=?", Waiting, sid)
	var check_sum string
	err1 := libs.DBSelectSingleColumn(&check_sum, "select check_sum from problems where problem_id=?", tinfo.Prob)
	if err != nil || err1 != nil {
		fmt.Println(err, err1)
		return false
	}

	for { //Repeating for data sync
		//get a new judger id
		judger.jid = libs.RandomString(64)
		res, err := http.Post(judger.url+"/judge?"+libs.GetQuerys(map[string]string{
			"mode": mode,
			"sum": check_sum,
			"cb": fmt.Sprintf(libs.BackDomain+"/FinishJudging?jid=%s", judger.jid),
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
			ProblemRLock(tinfo.Prob)
			file, err := os.Open(PRGetDataZip(tinfo.Prob))
			res, err1 = http.Post(judger.url+"/sync?"+libs.GetQuerys(map[string]string{"sum": check_sum}), "binary", file)
			ProblemRUnlock(tinfo.Prob)
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
	err = libs.DBSelectSingleColumn(&tinfo.Uuid, "select uuid from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if tinfo.Uuid == uuid {
		//Update status if and only if this is the recent submission
		go func() {
			err := SMUpdate(sid, tinfo.Prob, mode, ret)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}()
	}
	return true
}

func judgeCustomTest(sid int, callback *chan []byte, judger *Judger) {
	var content []byte
	err := libs.DBSelectSingleColumn(&content, "select content from custom_tests where id=?", sid)
	if err != nil {
		fmt.Println(err)
		*callback <- []byte{}
		return
	}
	res, err := http.Post(judger.url+"/custom?"+libs.GetQuerys(map[string]string{
		"cb": fmt.Sprintf(libs.BackDomain+"/FinishJudging?jid=%s", judger.jid),
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
	*callback <- <- judger.callback
}

func InsertSubmission(sid int, uuid int64, priority int, mode string) {
	waitingList.Push(JudgeEntry{sid, mode, uuid, nil}, priority)
}

func InsertCustomTest(sid int, callback *chan []byte) {
	waitingList.Push(JudgeEntry{sid, "custom_test", 0, callback}, 0)
}

func FinishJudging(ctx *gin.Context) {
	jid := ctx.Query("jid")
	result, _ := ioutil.ReadAll(ctx.Request.Body)
	for i := 0; i < 5; i++ {
		for key := range judgers {
			if judgers[key].jid == jid {
				judgers[key].callback <- result
				return
			}
		}
		time.Sleep(time.Second)
	}
	fmt.Printf("No such judger: judger_id=%s", jid)
}

func JudgerLog(ctx *gin.Context) {
	id := libs.GetIntDefault(ctx, "id", 0)
	if id >= len(judgers) {
		libs.APIWriteBack(ctx, 400, "no such judger", nil)
		return
	}
	res, err := http.Get(judgers[id].url + "/log")
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
		return
	}
	body, _ := ioutil.ReadAll(res.Body)
	libs.APIWriteBack(ctx, 200, "", map[string]any{"log": string(body)})
}