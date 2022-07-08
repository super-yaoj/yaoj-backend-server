package internal

import "yao/libs"

type CTStanding struct {
	TotalScore   float64
	TotalSScore  float64 //total sample score
	TotalPenalty int
	Scores       []float64
	SScores      []float64 //sample scores
	Penalties    []int
	UserId       int
	OrgRating    int
	NewRating    int
	PastContests int
}

var standings = map[int][]CTStanding{}

func CTRenewStanding(contest_id int) {
	var subs []struct {
		Id        int     `db:"submission_id" json:"submission_id"`
		Submitter int     `db:"submitter" json:"submitter"`
		Problem   int     `db:"problem_id"`
		Score  	  float64 `db:"score"`
	}
	libs.DBSelectAll(&subs, "select submission_id, submitter, problem_id, score from submissons where contest_id=?", contest_id)
	
}

func CTUpdateStanding() {

}