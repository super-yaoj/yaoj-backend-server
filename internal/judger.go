package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"yao/libs"

	"github.com/gin-gonic/gin"
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
	err := libs.DBSelectAll(&sub, "select submission_id, problem_id, contest_id from submissions where status < ? and status >= 0", Finished)
	if err != nil {
		log.Fatal(err)
	}
	for _, val := range sub {
		err := SMJudge(val.Id, val.ProblemId, val.ContestId)
		if err != nil {
			fmt.Println(err)
		}
	}
	for _, i := range judgers {
		go JudgerStart(i)
	}
}

type judgerResponse struct {
	Err      string `json:"error"`
	Err_code int    `json:"error_code"`
	Msg      string `json:"message"`
}

func JudgerStart(judger *Judger) {
	for {
		subm := waitingList.Pop().(JudgeEntry)
		sid, mode := subm.sid, subm.mode
		if mode != "custom_test" {
			if !JudgeSubmission(sid, mode, judger) {
				libs.DBUpdate("update submissions set status=? where submission_id=?", InternalError, sid)
			}
		} else {
			JudgeCustomTest(sid, subm.callback, judger)
		}
		//change the judger id to avoid attacks
		judger.jid = libs.RandomString(64)
	}
}

//return true or false indicates whether judging succeeds
func JudgeSubmission(sid int, mode string, judger *Judger) bool {
	go libs.DBUpdate("update submissions set status=status|? where submission_id=?", Waiting, sid)
	var content []byte
	err := libs.DBSelectSingleColumn(&content, "select content from submission_details where submission_id=?", sid)
	prob, err1 := libs.DBSelectSingleInt("select problem_id from submissions where submission_id=?", sid)
	var check_sum string
	err2 := libs.DBSelectSingleColumn(&check_sum, "select check_sum from problems where problem_id=?", prob)
	if err != nil || err1 != nil || err2 != nil {
		fmt.Println(err, err1, err2)
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
		json.Unmarshal(body, &jr)

		if jr.Msg == "ok" {
			break
		} else if jr.Err_code == 1 {
			ProblemRLock(prob)
			file, err := os.Open(PRGetDataZip(prob))
			res, err1 = http.Post(judger.url+"/sync?"+libs.GetQuerys(map[string]string{"sum": check_sum}), "binary", file)
			ProblemRUnlock(prob)
			if err != nil || err1 != nil {
				fmt.Printf("%v %v\n", err, err1)
				return false
			}
			body, _ = ioutil.ReadAll(res.Body)
			json.Unmarshal(body, &jr)
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
	go func() {
		err := SMUpdate(sid, prob, mode, ret)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}()
	return true
}

func JudgeCustomTest(sid int, callback *chan []byte, judger *Judger) {
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
	json.Unmarshal(body, &jr)
	if jr.Msg != "ok" {
		fmt.Println(jr.Err)
		*callback <- []byte{}
		return
	}
	*callback <- <- judger.callback
}

func InsertSubmission(sid, priority int, mode string) {
	waitingList.Push(JudgeEntry{sid, mode, nil}, priority)
}

func InsertCustomTest(sid int, callback *chan []byte) {
	waitingList.Push(JudgeEntry{sid, "custom_test", callback}, 0)
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