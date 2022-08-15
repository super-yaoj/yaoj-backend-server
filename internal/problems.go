package internal

import (
	"fmt"
	"sort"
	"yao/db"

	"github.com/super-yaoj/yaoj-core/pkg/problem"
	utils "github.com/super-yaoj/yaoj-utils"
)

// 钩子：数据库修改完成后，重新评测前
func OnProbRejudge(f func(int)) {
	Listen("OnProbRejudge", f)
}

type Statement struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type SubmConfig = problem.SubmConf // map[string]problem.SubmLimit

type Problem struct {
	Id    int    `db:"problem_id" json:"problem_id"`
	Title string `db:"title" json:"title"`
	Like  int    `db:"like" json:"like"`
	Liked bool   `json:"liked"`

	Statement_zh string `json:"statement_zh"`
	Statement_en string `json:"statement_en"`
	Tutorial_zh  string `json:"tutorial_zh"`
	Tutorial_en  string `json:"tutorial_en"`
	TimeLimit    int    `json:"time_limit"`
	MemoryLimit  int    `json:"memory_limit"`
	//Other statement can be seen by users
	HasSample  bool        `json:"has_sample"`
	Statements []Statement `json:"statements"`
	//01-string denotes which files can be downloaded
	AllowDown  string     `db:"allow_down" json:"allow_down"`
	SubmConfig SubmConfig `json:"subm_config"`
	//Could only be seen by admins
	DataInfo problem.DataInfo `json:"data"`
}

func ProbList(bound, pagesize, user_id int, isleft, isadmin bool) ([]Problem, bool, error) {
	pagesize += 1
	var problems []Problem
	if isadmin {
		var err error
		if isleft {
			err = db.SelectAll(&problems, "select problem_id, title, `like` from problems where problem_id>=? order by problem_id limit ?", bound, pagesize)
		} else {
			err = db.SelectAll(&problems, "select problem_id, title, `like` from problems where problem_id<=? order by problem_id desc limit ?", bound, pagesize)
		}
		if err != nil {
			return nil, false, err
		}
		isfull := len(problems) == pagesize
		if isfull {
			problems = problems[:pagesize-1]
		}
		if !isleft {
			utils.Reverse(problems)
		}
		ProbGetLikes(problems, user_id)
		return problems, isfull, nil
	} else {
		perms, err := UserPermissions(user_id)
		if err != nil {
			return nil, false, err
		}

		var ids []int
		perm_str := utils.JoinArray(perms)
		if isleft {
			ids, err = db.SelectInts(fmt.Sprintf("select distinct problem_id from problem_permissions where problem_id>=%d and permission_id in (%s) order by problem_id limit %d", bound, perm_str, pagesize))
		} else {
			ids, err = db.SelectInts(fmt.Sprintf("select distinct problem_id from problem_permissions where problem_id<=%d and permission_id in (%s) order by problem_id desc limit %d", bound, perm_str, pagesize))
		}
		if err != nil {
			return nil, false, err
		}

		isfull := len(ids) == pagesize
		if isfull {
			ids = ids[:pagesize-1]
		}
		if len(ids) != 0 {
			err = db.SelectAll(&problems, "select problem_id, title, `like` from problems where problem_id in ("+utils.JoinArray(ids)+")")
			if err != nil {
				return nil, false, err
			}
		}
		sort.Slice(problems, func(i, j int) bool { return problems[i].Id < problems[j].Id })
		ProbGetLikes(problems, user_id)
		return problems, isfull, nil
	}
}

func ProbGetLikes(problems []Problem, user_id int) {
	if user_id < 0 {
		return
	}
	ids := []int{}
	for _, i := range problems {
		ids = append(ids, i.Id)
	}
	ret := GetLikes(PROBLEM, user_id, ids)
	for i := range problems {
		problems[i].Liked = utils.HasInt(ret, problems[i].Id)
	}
}

func ProbQuery(problem_id, user_id int) (*Problem, error) {
	p := ProbLoad(problem_id)
	err := db.SelectSingle(&p, "select problem_id, title, `like`, allow_down from problems where problem_id=?", problem_id)
	if err != nil {
		return p, err
	}
	p.Liked = GetLike(PROBLEM, user_id, problem_id)
	return p, nil
}

func ProbGetPermissions(problem_id int) ([]Permission, error) {
	var p []Permission
	err := db.SelectAll(&p, "select a.permission_id, permission_name from ((select permission_id from problem_permissions where problem_id=? and permission_id>0) as a join permissions on a.permission_id=permissions.permission_id)", problem_id)
	return p, err
}

func ProbGetManagers(problem_id int) ([]User, error) {
	var u []User
	err := db.SelectAll(&u, "select user_id, user_name from ((select permission_id from problem_permissions where problem_id=? and permission_id<0) as a join user_info on -permission_id=user_id)", problem_id)
	return u, err
}

func ProbAddPermission(problem_id, permission_id int) error {
	_, err := db.Update("insert ignore into problem_permissions values (?, ?)", problem_id, permission_id)
	return err
}

func ProbDeletePermission(problem_id, permission_id int) error {
	_, err := db.Update("delete from problem_permissions where problem_id=? and permission_id=?", problem_id, permission_id)
	return err
}

// 数据库查询是否存在该题目
func ProbExists(problem_id int) bool {
	count, _ := db.SelectSingleInt("select count(*) from problems where problem_id=?", problem_id)
	return count > 0
}

func ProbRejudge(problem_id int) error {
	current := utils.TimeStamp()
	var sub []SubmissionBase
	err := db.SelectAll(&sub, "select submission_id, contest_id from submissions where problem_id=?", problem_id)
	if err != nil {
		return err
	}
	//update uuid to current time-stamp
	_, err = db.Update("update submissions set uuid=?, status=0 where problem_id=?", current, problem_id)
	if err != nil {
		return err
	}
	Register("OnProbRejudge", problem_id)
	for _, i := range sub {
		SubmJudge(i, true, current)
	}
	return nil
}
