package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"yao/libs"
)

func insert() {
	libs.DBInit()
	cmd := ""
	for i := 0; i < 1000000; i++ {
		//user cur := fmt.Sprintf("(null, \"%s\", \"%s\", \"\", 0, now(), \"\", 2, 0, \"\", \"\")", libs.RandomString(30), libs.RandomString(64))
		//blogs cur := fmt.Sprintf("(null, %d, \"%s\", \"%s\", %d, now())", rand.Intn(5000) + 1, libs.RandomString(100), libs.RandomString(1000), rand.Intn(2))
		//submission cur := fmt.Sprintf("(null, %d, %d, %d, %d, 100, 1000, 1000, 1, now(), \"%s\", \"%s\")", rand.Intn(5000), rand.Intn(5000), rand.Intn(100), rand.Intn(7), libs.RandomString(1000), libs.RandomString(1000))
		cur := fmt.Sprintf("(%d, %d)", rand.Intn(100000), rand.Intn(10000))
		if i % 100 == 0 {
			cmd = "insert ignore into user_permissions values " + cur
		} else {
			cmd += "," + cur
		}
		if i % 100 == 99 {
			_, err := libs.DBUpdate(cmd)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		if i % 10000 == 0 {
			fmt.Println(i)
		}
	}
}

type SubmissionQ struct {
	Submission_id int
	Permission_id int
	Score float32
	Submitter int
	Language int
	State int
	Time_used int
	Memory_used int
	Code_length int
	Submit_time time.Time
	Result string
}

type SubmissionQA []SubmissionQ
func (this SubmissionQA) Len() int {
	return len(this)
}

func (this SubmissionQA) Less(i, j int) bool {
	return this[i].Submission_id < this[j].Submission_id
}

func (this SubmissionQA) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func query() {
	fmt.Println("Preparing...")
	rand.Seed(time.Now().Unix())
	numbers := strings.Builder{}
	numbers.WriteString("1")
	for i := 0; i < 1000; i++ {
		numbers.WriteString(fmt.Sprintf(",%d", rand.Intn(200000)))
	}
	fmt.Println("Begin...")
	var submissions SubmissionQ
	beg := time.Now().UnixMicro()
	libs.DBSelectSingle(&submissions, "select submission_id, result from submissions where (contest_id in (" + numbers.String() + ")) or (problem_id in (" + numbers.String() + ")) or submitter=1 order by submission_id desc limit 100")
	// libs.DBSelectInts("select distinct problem_id from problem_permissions where problem_id>=1 and permission_id in (" + numbers + ") order by problem_id limit 100")
	// libs.DBQuery("select problem_id, title, `like` from problems where problem_id<=? order by problem_id desc limit 100", 100000)
	fmt.Println(submissions)
	fmt.Println(time.Now().UnixMicro() - beg)
}

func main() {
	libs.DBInit()
	begin := time.Now().UnixMicro()
	var ret string
	libs.DBSelectSingleColumn(&ret, "select result from submissions where submission_id=?", 3)
	fmt.Println(ret)
	fmt.Println(time.Now().UnixMicro() - begin)
	libs.DBClose()
}