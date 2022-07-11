package internal

import (
	"fmt"
	"time"
	"yao/libs"

	jsoniter "github.com/json-iterator/go"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
)

type CTStandingEntry struct {
	UserId       int
	TotalScore   float64
	TotalSScore  float64 //total sample score
	TotalPenalty time.Duration
	TotalHacked  bool
	SubIds 		 []int
	Scores       []float64
	SScores      []float64 //sample scores
	Penalties    []time.Duration
	Hacked 		 []bool
	OrgRating    int
	//these two below only be used at rating calculation
	NewRating    int
	PastContests int
}

type CTStanding struct {
	entries []CTStandingEntry
	uidMap, pidMap  map[int]int
	startTime time.Time
}

type standingSubm struct {
	Id        	int     	`db:"submission_id" json:"submission_id"`
	Submitter 	int     	`db:"submitter" json:"submitter"`
	Problem   	int     	`db:"problem_id"`
	Score  	  	float64 	`db:"score"`
	SampleScore float64 	`db:"sample_score"`
	Penalty   	time.Time 	`db:"submit_time"`
	Hacked 		bool 		`db:"hacked"`
}

var (
	allStandings = libs.NewMemoryCache(time.Hour, 100)
	ctsMultiLock = libs.NewMappedMultiRWMutex()
	dataColumns = "submission_id, submitter, problem_id, score, sample_score, hacked, submit_time"
)

func newStandingEntry(userid, userating, probs int) CTStandingEntry {
	return CTStandingEntry{
		userid, 0, 0, 0, false,
		make([]int, probs),
		make([]float64, probs),
		make([]float64, probs),
		make([]time.Duration, probs),
		make([]bool, probs),
		userating, 0, 0,
	}
}

func updateEntry(standing *CTStanding, sub *standingSubm, getRating bool) {
	uid, ok := standing.uidMap[sub.Submitter]
	if !ok {
		uid = len(standing.entries)
		standing.uidMap[sub.Submitter] = uid
		rating := 0
		if getRating {
			rating, _ = libs.DBSelectSingleInt("select rating from user_info where user_id=?", sub.Submitter)
		}
		standing.entries = append(standing.entries, newStandingEntry(sub.Submitter, rating, len(standing.pidMap)))
	}
	pid, ok := standing.pidMap[sub.Problem]
	if !ok {
		fmt.Println("No such contest problem in CTRenewStanding()!!")
		return
	}
	entry := &standing.entries[uid]
	if entry.SubIds[pid] > sub.Id {
		return
	}
	entry.Scores[pid] = sub.Score
	entry.SScores[pid] = sub.SampleScore
	entry.SubIds[pid] = sub.Id
	entry.Penalties[pid] = sub.Penalty.Sub(standing.startTime)
	entry.Hacked[pid] = sub.Hacked
	entry.TotalScore = 0
	for _, i := range entry.Scores {
		entry.TotalScore += i
	}
	entry.TotalSScore = 0
	for _, i := range entry.SScores {
		entry.TotalSScore += i
	}
	entry.TotalPenalty = 0
	for _, i := range entry.Penalties {
		entry.TotalPenalty += i
	}
	entry.TotalHacked = false
	for _, i := range entry.Hacked {
		entry.TotalHacked = entry.TotalHacked || i
	}
}

func CTRenewStanding(contest_id int) {
	ctsMultiLock.Lock(contest_id)
	defer ctsMultiLock.Unlock(contest_id)
	contest, err := CTQuery(contest_id, -1)
	if err != nil {
		fmt.Println(err)
		return
	}
	var subs []standingSubm
	err = libs.DBSelectAll(&subs, "select " + dataColumns + " from submissons where contest_id=? order by submission_id", contest_id)
	if err != nil {
		fmt.Println(err)
		return
	}
	allStandings.Delete(contest_id)
	standing := &CTStanding {
		startTime: contest.StartTime,
		entries: make([]CTStandingEntry, 0),
		uidMap: make(map[int]int),
		pidMap: make(map[int]int),
	}
	probs, err := CTGetProblems(contest_id)
	if err != nil {
		fmt.Println(err)
		return
	}
	for key, val := range probs {
		standing.pidMap[val.Id] = key
	}
	
	for _, sub := range subs {
		updateEntry(standing, &sub, false)
	}
	uids := make([]int, len(subs))
	for i := range subs {
		uids[i] = subs[i].Submitter
	}
	rows, err := libs.DBQuery("select user_id, rating from user_info where user_id in (" + libs.JoinArray(uids) + ")")
	defer rows.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	var user_rating map[int]int
	for rows.Next() {
		var id, rating int
		rows.Scan(&id, &rating)
		user_rating[id] = rating
	}
	for i := range standing.entries {
		standing.entries[i].OrgRating = user_rating[standing.entries[i].UserId]
	}
	allStandings.Set(contest_id, standing)
}

func CTSUpdateSubmission(contest_id, sid int) {
	ctsMultiLock.Lock(contest_id)
	defer ctsMultiLock.Unlock(contest_id)
	standing, ok := allStandings.Get(contest_id)
	if !ok {
		ctsMultiLock.Unlock(contest_id)
		CTRenewStanding(contest_id)
		return
	}
	var sub standingSubm
	err := libs.DBSelectSingle(&sub, "select " + dataColumns + " from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return
	}
	updateEntry(standing.(*CTStanding), &sub, true)
}

func CTSDeleteSubmission(sub SubmissionBase) {
	ctsMultiLock.Lock(sub.ContestId)
	defer ctsMultiLock.Unlock(sub.ContestId)
	raw_standing, ok := allStandings.Get(sub.ContestId)
	if !ok {
		ctsMultiLock.Unlock(sub.ContestId)
		CTRenewStanding(sub.ContestId)
		return
	}
	standing := raw_standing.(*CTStanding)
	uid := standing.uidMap[sub.Submitter]
	pid := standing.pidMap[sub.ProblemId]
	if standing.entries[uid].SubIds[pid] > sub.Id {//isn't the last commit
		return
	}
	var newsub standingSubm
	err := libs.DBSelectSingle(&sub, "select " + dataColumns + " from submissions where problem_id=? and contest_id=? and submitter=? order by submission_id desc limit 1", sub.ProblemId, sub.ContestId, sub.Submitter)
	if err != nil {//no more submissions
		newsub = standingSubm{Submitter: sub.Submitter, Problem: sub.ProblemId}
	}
	updateEntry(standing, &newsub, false)
}

func CTGetStanding(contest_id int) []CTStandingEntry {
	ctsMultiLock.RLock(contest_id)
	defer ctsMultiLock.RUnlock(contest_id)
	standing, ok := allStandings.Get(contest_id)
	if !ok {
		finished, err := libs.DBSelectSingleInt("select finished from contests where contest_id=?", contest_id)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if finished > 0 {
			var entries []CTStandingEntry
			var js []byte
			err = libs.DBSelectSingleColumn(&js, "select standing from contest_standing where contest_id=?", contest_id)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			err = jsoniter.Unmarshal(js, &entries)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			return entries
		}
		ctsMultiLock.RUnlock(contest_id)
		CTRenewStanding(contest_id)
		ctsMultiLock.RLock(contest_id)
		standing, ok = allStandings.Get(contest_id)
		if !ok {
			fmt.Println("Reading contest standing error")
			return nil
		}
	}
	return standing.(*CTStanding).entries
}

func (entry CTStandingEntry) Rate(rating int) {
	entry.NewRating = rating
}
func (entry CTStandingEntry) Rating() int {
	return entry.OrgRating
}
func (entry CTStandingEntry) Count() int {
	return entry.PastContests
}

func getPastContests(entries []CTStandingEntry) error {
	uids := []int{}
	for i := range entries {
		uids = append(uids, entries[i].UserId)
	}
	rows, err := libs.DBQuery("select user_id, count(*) from ratings where user_id in (" + libs.JoinArray(uids) + ") group by user_id")
	defer rows.Close()
	if err != nil {
		return err
	}
	var ratings map[int]int
	for rows.Next() {
		var uid, count int
		rows.Scan(&uid, &count)
		ratings[uid] = count
	}
	for i := range entries {
		entries[i].PastContests = ratings[entries[i].UserId]
	}
	return nil
}
/*
You must ensure that there's no more submissions judging in this contest.
*/
func CTFinish(contest_id int) error {
	standing := CTGetStanding(contest_id)
	err := getPastContests(standing)
	if err != nil {
		return err
	}
	err = utils.CalcRating(standing)
	if err != nil {
		return err
	}
	//Write standings into database
	js, err := jsoniter.Marshal(standing)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("insert into contest_standing values (?, ?)", contest_id, js)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("update contests set finished=1 where contest_id=?", contest_id)
	return err
}