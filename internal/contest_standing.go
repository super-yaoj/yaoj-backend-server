package internal

import (
	"fmt"
	"time"
	"yao/db"

	jsoniter "github.com/json-iterator/go"
	utils "github.com/super-yaoj/yaoj-utils"
	"github.com/super-yaoj/yaoj-utils/cache"
	"github.com/super-yaoj/yaoj-utils/locks"
	"github.com/super-yaoj/yaoj-utils/ratings"
)

type CTStandingEntry struct {
	UserId       int
	UserName 	 string
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
	Accepted 	int 	`db:"accepted"`
}

type standingUser struct {
	Rating 		int 	`db:"rating"`
	UserName 	string 	`db:"user_name"`
}

var (
	allStandings = cache.NewMemoryCache[*CTStanding](time.Hour, 100)
	ctsMultiLock = locks.NewMappedMultiRWMutex()
	standingCols = "submission_id, submitter, problem_id, score, sample_score, accepted, submit_time"
)

func init() {
	AfterSubmJudge(func(sb SubmissionBase) {
		if sb.ContestId > 0 {
			ctsUpdateSubmission(sb.ContestId, sb.Id)
		}
	})
	AfterSubmDelete(func(sb SubmissionBase) {
		if sb.ContestId > 0 {
			ctsDeleteSubmission(sb)
		}
	})
	AfterCTModify(func(i int) {
		ctsRenew(i)
	})
}

func newStandingEntry(user_id, rating int, user_name string, probs int) CTStandingEntry {
	return CTStandingEntry{
		user_id, user_name,
		make([]int, probs),
		make([]float64, probs),
		make([]float64, probs),
		make([]time.Duration, probs),
		make([]bool, probs),
		rating, 0, 0,
	}
}

func updateCTSEntry(standing *CTStanding, sub *standingSubm, getRating bool) {
	uid, ok := standing.uidMap[sub.Submitter]
	if !ok {
		uid = len(standing.entries)
		standing.uidMap[sub.Submitter] = uid
		info := standingUser{}
		if getRating {
			db.DBSelectSingle(&info, "select rating, user_name from user_info where user_id=?", sub.Submitter)
		}
		standing.entries = append(standing.entries, newStandingEntry(sub.Submitter, info.Rating, info.UserName, len(standing.pidMap)))
	}
	pid, ok := standing.pidMap[sub.Problem]
	if !ok {
		fmt.Println("No such contest problem in updateCTSEntry()!!")
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
	entry.Hacked[pid] = (sub.Accepted & ExtraAccepted) == 0
}

func ctsRenew(contest_id int) {
	ctsMultiLock.Lock(contest_id)
	defer ctsMultiLock.Unlock(contest_id)
	contest, err := CTQuery(contest_id, -1)
	if err != nil {
		fmt.Println(err)
		return
	}
	var subs []standingSubm
	err = db.DBSelectAll(&subs, "select " + standingCols + " from submissions where contest_id=? order by submission_id", contest_id)
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
	
	if len(subs) > 0 {
		for _, sub := range subs {
			updateCTSEntry(standing, &sub, false)
		}
		uids := make([]int, len(subs))
		for i := range subs {
			uids[i] = subs[i].Submitter
		}
		rows, err := db.DBQuery("select user_id, rating, user_name from user_info where user_id in (" + utils.JoinArray(uids) + ")")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer rows.Close()
		user_rating := make(map[int]standingUser)
		for rows.Next() {
			var id, rating int
			var user_name string
			rows.Scan(&id, &rating, &user_name)
			user_rating[id] = standingUser{rating, user_name}
		}
		for i := range standing.entries {
			user := user_rating[standing.entries[i].UserId]
			standing.entries[i].UserName  = user.UserName
			standing.entries[i].OrgRating = user.Rating
		}
	}
	allStandings.Set(contest_id, standing)
}

func ctsUpdateSubmission(contest_id, sid int) {
	if CTHasFinished(contest_id) {
		return
	}
	ctsMultiLock.Lock(contest_id)
	defer ctsMultiLock.Unlock(contest_id)
	standing, ok := allStandings.Get(contest_id)
	if !ok {
		return
	}
	var sub standingSubm
	err := db.DBSelectSingle(&sub, "select " + standingCols + " from submissions where submission_id=?", sid)
	if err != nil {
		fmt.Println(err)
		return
	}
	updateCTSEntry(standing, &sub, true)
}

func ctsDeleteSubmission(sub SubmissionBase) {
	if CTHasFinished(sub.ContestId) {
		return
	}
	ctsMultiLock.Lock(sub.ContestId)
	defer ctsMultiLock.Unlock(sub.ContestId)
	standing, ok := allStandings.Get(sub.ContestId)
	if !ok {
		return
	}
	uid := standing.uidMap[sub.Submitter]
	pid := standing.pidMap[sub.ProblemId]
	if standing.entries[uid].SubIds[pid] > sub.Id {//isn't the last commit
		return
	}
	standing.entries[uid].SubIds[pid] = 0
	var newsub standingSubm
	err := db.DBSelectSingle(&sub, "select " + standingCols + " from submissions where problem_id=? and contest_id=? and submitter=? order by submission_id desc limit 1", sub.ProblemId, sub.ContestId, sub.Submitter)
	if err != nil {//no more submissions
		newsub = standingSubm{Submitter: sub.Submitter, Problem: sub.ProblemId}
	}
	updateCTSEntry(standing, &newsub, false)
}

func CTSGet(contest_id int) []CTStandingEntry {
	ctsMultiLock.RLock(contest_id)
	defer ctsMultiLock.RUnlock(contest_id)
	standing, ok := allStandings.Get(contest_id)
	if !ok {
		if CTHasFinished(contest_id) {
			var entries []CTStandingEntry
			var js []byte
			err := db.DBSelectSingleColumn(&js, "select standing from contest_standing where contest_id=?", contest_id)
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
		ctsRenew(contest_id)
		ctsMultiLock.RLock(contest_id)
		standing, ok = allStandings.Get(contest_id)
		if !ok {
			fmt.Println("Reading contest standing error")
			return nil
		}
	}
	return standing.entries
}

func (entry *CTStandingEntry) Rate(rating int) {
	entry.NewRating = rating
}
func (entry *CTStandingEntry) Rating() int {
	return entry.OrgRating
}
func (entry *CTStandingEntry) Count() int {
	return entry.PastContests
}

func getPastContests(entries []CTStandingEntry) error {
	uids := []int{}
	for i := range entries {
		uids = append(uids, entries[i].UserId)
	}
	rows, err := db.DBQuery("select user_id, count(*) from ratings where user_id in (" + utils.JoinArray(uids) + ") group by user_id")
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
	var err error
	//For safety, recalculate standing
	ctsRenew(contest_id)
	standing := CTSGet(contest_id)
	if len(standing) > 0 {
		err = getPastContests(standing)
		if err != nil {
			return err
		}
		standing_p := make([]*CTStandingEntry, len(standing))
		for i := range standing {
			standing_p[i] = &standing[i]
		}
		err = ratings.CalcRating(standing_p)
		if err != nil {
			return err
		}
		//Write standings into database
		uids := make([]int, len(standing))
		values := make([]string, len(standing))
		for key, i := range standing {
			uids[key] = i.UserId
		}
		//save rating changes to table `ratings`
		current := time.Now().UTC().Format("2006-01-02 15:04:05")
		for key, i := range standing {
			values[key] = fmt.Sprintf("(%d, %d, %d, \"%s\")", i.UserId, i.NewRating, contest_id, current)
		}
		_, err = db.DBUpdate("insert into ratings values " + utils.JoinArray(values))
		if err != nil {
			return err
		}
		//change user rating in table `user_info`
		for key, i := range standing {
			values[key] = fmt.Sprintf("(%d, %d)", i.UserId, i.NewRating)
		}
		_, err = db.DBUpdate("insert into user_info (user_id, rating) values " + utils.JoinArray(values) + " on duplicate key update rating=values(rating)")
		if err != nil {
			return err
		}
	}
	js, err := jsoniter.Marshal(standing)
	if err != nil {
		return err
	}
	_, err = db.DBUpdate("insert into contest_standing values (?, ?)", contest_id, js)
	if err != nil {
		return err
	}
	_, err = db.DBUpdate("update contests set finished=1 where contest_id=?", contest_id)
	return err
}