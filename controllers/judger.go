package controllers

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
	//The submission which this judger is running on, 0 means free
	submission 	int
	callback 	chan []byte
}

func NewJudger(url string) *Judger {
	return &Judger{ url, 0, make(chan []byte) }
}

var judgers = []*Judger {
	NewJudger("http://localhost:2375"),
	// "http://localhost:8083",
	// "http://localhost:8084",
}

var waitingList = libs.NewBlockPriorityQueue()

const (
	Ok int = iota
	CompileError
	Waiting
	Judging
	SystemError
)

func JudgersInit() {
	ids, err := libs.DBSelectInts("select submission_id from submissions where status in (?, ?)", Waiting, Judging)
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range ids {
		InsertSubmission(i, 1)
	}
	for _, i := range judgers {
		go JudgerStart(i)
	}
}

type judgerResponse struct {
	Err 		string 	`json:"error"`
	Err_code 	int 	`json:"error_code"`
	Msg 		string 	`json:"message"`
}

func JudgerStart(judger *Judger) {
	for {
		sid := waitingList.Pop().(int)
		go libs.DBUpdate("update submissions set status=? where submission_id=?", Judging, sid)
		
		var content []byte
		err := libs.DBSelectSingleColumn(&content, "select content from submission_content where submission_id=?", sid)
		prob, err1 := libs.DBSelectSingleInt("select problem_id from submissions where submission_id=?", sid)
		var check_sum string
		err2 := libs.DBSelectSingleColumn(&check_sum, "select check_sum from problems where problem_id=?", prob)
		if err != nil || err1 != nil || err2 != nil {
			fmt.Printf("%v %v %v", err, err1, err2)
			libs.DBUpdate("update submissions set status=? where submission_id=?", SystemError, sid)
			continue
		}
		
		failed := true
		for {//Repeating for data sync
			res, err := http.Post(judger.url + "/judge?" + libs.GetQuerys(map[string]string{
				"sum": check_sum,
				//Give a check_sum of submission_id for security
				"cb": fmt.Sprintf(libs.BackDomain + "/FinishJudging?submission_id=%d&check_sum=%s", sid, SaltPassword(fmt.Sprint(sid))),
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
				res, err1 = http.Post(judger.url + "/sync?" + libs.GetQuerys(map[string]string{ "sum": check_sum }), "binary", file)
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
			libs.DBUpdate("update submissions set status=? where submission_id=?", SystemError, sid)
			continue
		}
		//Waiting judger finishes
		judger.submission = sid
		err = SMUpdate(sid, prob, <-judger.callback)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		judger.submission = 0
	}
}

func InsertSubmission(sid, priority int) {
	waitingList.Push(sid, priority)
}

func judgerCallback(sid int, result []byte) {
	for i := 0; i < 5; i++ {
		for key := range judgers {
			if judgers[key].submission == sid {
				judgers[key].callback <- result
				return
			}
		}
		time.Sleep(time.Second)
	}
	fmt.Printf("No such judger: submission_id=%d", sid)
}

func FinishJudging(ctx *gin.Context) {
	sid, ok := libs.GetInt(ctx, "submission_id")
	if !ok {
		fmt.Printf("Judge failed: no submission_id field\n")
		return
	}
	sum := ctx.Query("check_sum")
	if sum != SaltPassword(fmt.Sprint(sid)) {
		libs.APIWriteBack(ctx, 400, "", nil)
		fmt.Printf("Judge failed: check sum of submission_id error\n")
		return
	}
	body, _ := ioutil.ReadAll(ctx.Request.Body)
	judgerCallback(sid, body)
}