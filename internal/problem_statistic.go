package internal

import (
	"fmt"
	"math"
	"sort"
	"time"
	"yao/db"

	utils "github.com/super-yaoj/yaoj-utils"
	"github.com/super-yaoj/yaoj-utils/cache"
	"github.com/super-yaoj/yaoj-utils/locks"
)

type statisticValue struct {
	value  []int
	sid    []int
	sorted []int
}

type problemStatistic struct {
	uidMap  map[int]int
	sids    map[int]struct{}
	totSubs int
	time    statisticValue
	memory  statisticValue
	length  statisticValue
}

type statisticSubm struct {
	Id        	int `db:"submission_id"`
	Submitter 	int `db:"submitter"`
	Problem   	int `db:"problem_id"`
	Time        int `db:"time"`
	Memory      int `db:"memory"`
	Length      int `db:"length"`
}

var (
	allStatistic = cache.NewMemoryCache[*problemStatistic](time.Hour, 100)
	prsMultiLock = locks.NewMappedMultiRWMutex()
	statisticCols = "submission_id, submitter, problem_id, time, memory, length"
)

func init() {
	AfterSubmCreate(func(sb SubmissionBase) {
		prsAddSubmission(sb.ProblemId, sb.Id)
	})
	AfterSubmDelete(func(sb SubmissionBase) {
		prsDeleteSubmission(sb)
	})
	AfterSubmJudge(func(sb SubmissionBase) {
		prsUpdateSubmission(sb.ProblemId, sb.Id)
	})
	OnSubmRejudge(func (sb SubmissionBase) {
		prsDeleteSubmission(sb)
	})
	OnProbRejudge(func(i int) {
		prsRenew(i)
	})
}

func (val *statisticValue) compare() func(int, int) bool {
	return func(i, j int) bool {
		a, b := val.sorted[i], val.sorted[j]
		if val.value[a] == val.value[b] {
			return val.sid[a] < val.sid[b]
		}
		return val.value[a] < val.value[b]
	}
}

func (val *statisticValue) newEntry(uid int) {
	val.value = append(val.value, math.MaxInt)
	val.sid = append(val.sid, 0)
	val.sorted = append(val.sorted, uid)
}

func (val *statisticValue) initEntry(uid int) {
	val.value[uid], val.sid[uid] = math.MaxInt, 0
}

func (val *statisticValue) resortEntry(uid int) {
	utils.ResortEntry(val.sorted, val.compare(), utils.FindIndex(val.sorted, uid))
}

func (val *statisticValue) updateEntry(value int, sid int, uid int, sorts bool) {
	if val.value[uid] > value {
		val.value[uid], val.sid[uid] = value, sid
		if sorts {
			val.resortEntry(uid)
		}
	}
}

func (statistic *problemStatistic) newEntry(user_id int) int {
	uid := len(statistic.uidMap)
	statistic.uidMap[user_id] = uid
	statistic.time.newEntry(uid)
	statistic.memory.newEntry(uid)
	statistic.length.newEntry(uid)
	return uid
}

func (statistic *problemStatistic) updateEntry(sub *statisticSubm, sorts bool) {
	uid, ok := statistic.uidMap[sub.Submitter]
	if !ok {
		uid = statistic.newEntry(sub.Submitter)
	}
	statistic.time.updateEntry(sub.Time, sub.Id, uid, sorts)
	statistic.memory.updateEntry(sub.Memory, sub.Id, uid, sorts)
	statistic.length.updateEntry(sub.Length, sub.Id, uid, sorts)
}

func prsRenew(problem_id int) {
	prsMultiLock.Lock(problem_id)
	defer prsMultiLock.Unlock(problem_id)
	var subs []statisticSubm
	err := db.DBSelectAll(&subs, "select " + statisticCols + " from submissions where problem_id=? and status=? and accepted=?", problem_id, Finished, Accepted)
	if err != nil {
		fmt.Println(err)
		return
	}
	allStatistic.Delete(problem_id)
	statistic := &problemStatistic{}
	statistic.uidMap = make(map[int]int)
	statistic.sids = make(map[int]struct{})
	statistic.totSubs, err = db.DBSelectSingleInt("select count(*) from submissions where problem_id=?", problem_id)
	for i := range subs {
		statistic.sids[subs[i].Id] = struct{}{}
		statistic.updateEntry(&subs[i], false)
	}
	sort.Slice(statistic.time.sorted, statistic.time.compare())
	sort.Slice(statistic.memory.sorted, statistic.memory.compare())
	sort.Slice(statistic.length.sorted, statistic.length.compare())
	allStatistic.Set(problem_id, statistic)
}

func prsAddSubmission(problem_id, sid int) {
	prsMultiLock.Lock(problem_id)
	defer prsMultiLock.Unlock(problem_id)
	statistic, ok := allStatistic.Get(problem_id)
	if !ok {
		return
	}
	statistic.totSubs++
}

func prsUpdateSubmission(problem_id, sid int) {
	prsMultiLock.Lock(problem_id)
	defer prsMultiLock.Unlock(problem_id)
	statistic, ok := allStatistic.Get(problem_id)
	if !ok {
		return
	}
	var sub statisticSubm
	err := db.DBSelectSingle(&sub, "select " + statisticCols + " from submissions where submission_id=? and status=? and accepted=?", sid, Finished, Accepted)
	if err != nil {
		//this submission isn't ac
		return
	}
	statistic.sids[sid] = struct{}{}
	statistic.updateEntry(&sub, true)
}

func prsDeleteSubmission(sub SubmissionBase) {
	prsMultiLock.Lock(sub.ProblemId)
	defer prsMultiLock.Unlock(sub.ProblemId)
	statistic, ok := allStatistic.Get(sub.ProblemId)
	if !ok {
		return
	}
	_, ok = statistic.sids[sub.Id]
	statistic.totSubs--
	if !ok {//no such ac submission
		return
	}
	delete(statistic.sids, sub.Id)
	uid, ok := statistic.uidMap[sub.Submitter]
	var subs []statisticSubm
	err := db.DBSelectAll(&subs, "select " + statisticCols + " from submissions where problem_id=? and submitter=? and status=? and accepted=?", sub.ProblemId, sub.Submitter, Finished, Accepted)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(subs) == 0 {
		if !ok {
			return
		}
		allStatistic.Delete(sub.ProblemId)
		return
	}
	if !ok {
		uid = statistic.newEntry(sub.Submitter)
	} else {
		statistic.time.initEntry(uid)
		statistic.memory.initEntry(uid)
		statistic.length.initEntry(uid)
	}
	for i := range subs {
		statistic.updateEntry(&subs[i], false)
	}
	statistic.time.resortEntry(uid)
	statistic.memory.resortEntry(uid)
	statistic.length.resortEntry(uid)
}

/*
mode is one of {"time", "memory"}
return (submission ids, is full)
*/
func ProbSGetSubmissions(problem_id, bound, bound_id, pagesize int, isleft bool, mode string) ([]int, bool) {
	prsMultiLock.RLock(problem_id)
	defer prsMultiLock.RUnlock(problem_id)
	statistic, ok := allStatistic.Get(problem_id)
	if !ok {
		prsMultiLock.RUnlock(problem_id)
		prsRenew(problem_id)
		prsMultiLock.RLock(problem_id)
		statistic, ok = allStatistic.Get(problem_id)
		if !ok {
			return nil, false
		}
	}
	n := len(statistic.uidMap)
	val := &statistic.time
	if mode == "memory" {
		val = &statistic.memory
	} else if mode == "length" {
		val = &statistic.length
	}
	
	start := sort.Search(n, func(i int) bool {
		uid := val.sorted[i]
		return val.value[uid] > bound || (val.value[uid] == bound && utils.If(isleft, val.sid[uid] >= bound_id, val.sid[uid] > bound_id))
	})
	subs := []int{}
	if isleft {
		for i := start; i < n && i < start + pagesize; i++ {
			subs = append(subs, val.sid[val.sorted[i]])
		}
		return subs, start + pagesize + 1 <= n
	} else {
		for i := utils.Max(0, start - pagesize); i < start; i++ {
			subs = append(subs, val.sid[val.sorted[i]])
		}
		return subs, start > pagesize
	}
}

func ProbSGetACRatio(problem_id int) (int, int) {
	prsMultiLock.RLock(problem_id)
	defer prsMultiLock.RUnlock(problem_id)
	statistic, ok := allStatistic.Get(problem_id)
	if !ok {
		prsMultiLock.RUnlock(problem_id)
		prsRenew(problem_id)
		prsMultiLock.RLock(problem_id)
		statistic, ok = allStatistic.Get(problem_id)
		if !ok {
			return 0, 0
		}
	}
	return len(statistic.sids), statistic.totSubs
}