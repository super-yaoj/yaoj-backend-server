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
	url string
	//unique random judger id, change after each request
	jid string
	callback   chan []byte
}

type JudgeEntry struct {
	sid int
	mode string //one of "pretest", "tests", "extra"
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
	err := libs.DBSelectAll(&sub, "select submission_id, problem_id, contest_id from submissions where status < ?", Finished)
	if err != nil {
		log.Fatal(err)
	}
	for _, val := range sub {
		err := JudgeSubmission(val.Id, val.ProblemId, val.ContestId)
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
		go libs.DBUpdate("update submissions set status=status|? where submission_id=?", Waiting, sid)

		var content []byte
		err := libs.DBSelectSingleColumn(&content, "select content from submission_details where submission_id=?", sid)
		prob, err1 := libs.DBSelectSingleInt("select problem_id from submissions where submission_id=?", sid)
		var check_sum string
		err2 := libs.DBSelectSingleColumn(&check_sum, "select check_sum from problems where problem_id=?", prob)
		if err != nil || err1 != nil || err2 != nil {
			fmt.Printf("%v %v %v", err, err1, err2)
			libs.DBUpdate("update submissions set status=? where submission_id=?", InternalError, sid)
			continue
		}

		failed := true
		for { //Repeating for data sync
			//get a new judger id
			judger.jid = libs.RandomString(64)
			res, err := http.Post(judger.url+"/judge?"+libs.GetQuerys(map[string]string{
				"mode": mode,
				"sum": check_sum,
				//Give a check_sum of submission_id for security
				"cb": fmt.Sprintf(libs.BackDomain+"/FinishJudging?jid=%s", judger.jid),
			}), "binary", bytes.NewBuffer(content))
			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}
			body, _ := ioutil.ReadAll(res.Body)
			var jr judgerResponse
			json.Unmarshal(body, &jr)

			if jr.Msg == "ok" {
				failed = false
				break
			} else if jr.Err_code == 1 {
				ProblemRLock(prob)
				file, err := os.Open(PRGetDataZip(prob))
				res, err1 = http.Post(judger.url+"/sync?"+libs.GetQuerys(map[string]string{"sum": check_sum}), "binary", file)
				ProblemRUnlock(prob)
				if err != nil || err1 != nil {
					fmt.Printf("%v %v\n", err, err1)
					break
				}
				body, _ = ioutil.ReadAll(res.Body)
				json.Unmarshal(body, &jr)
				if jr.Msg != "ok" {
					fmt.Printf("%s\n", jr.Err)
					break
				}
			} else {
				fmt.Printf("%s\n", jr.Err)
				break
			}
		}
		if failed {
			libs.DBUpdate("update submissions set status=? where submission_id=?", InternalError, sid)
			continue
		}
		//Waiting judger finishes
		ret := <-judger.callback
		go func() {
			err = SMUpdate(sid, prob, mode, ret)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}()
		//change the judger id to avoid attacks
		judger.jid = libs.RandomString(64)
	}
}

func InsertSubmission(sid, priority int, mode string) {
	waitingList.Push(JudgeEntry{sid, mode}, priority)
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
