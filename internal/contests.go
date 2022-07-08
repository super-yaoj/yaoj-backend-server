package internal

import (
	"fmt"
	"sort"
	"time"
	"yao/libs"
)

type Contest struct {
	Id             int       `db:"contest_id" json:"contest_id"`
	Title          string    `db:"title" json:"title"`
	StartTime      time.Time `db:"start_time" json:"start_time"`
	EndTime        time.Time `db:"end_time" json:"end_time"`
	Pretest        bool      `db:"pretest" json:"pretest"`
	ScorePrivate   bool      `db:"score_private" json:"score_private"`
	Finished       bool      `db:"finished" json:"finished"`
	Like           int       `db:"like" json:"like"`
	Liked          bool      `json:"liked"`
	RegisterStatus int       `json:"register_status"`
	Registrants    int       `db:"registrants" json:"registrants"`
}

func CTList(bound, pagesize, user_id int, isleft, isadmin bool) ([]Contest, bool, error) {
	pagesize += 1
	var cts []Contest
	if isadmin {
		var err error
		if isleft {
			err = libs.DBSelectAll(&cts, "select * from contests where contest_id<=? order by contest_id desc limit ?", bound, pagesize)
		} else {
			err = libs.DBSelectAll(&cts, "select * from contests where contest_id>=? order by contest_id limit ?", bound, pagesize)
		}
		if err != nil {
			return nil, false, err
		}
		isfull := len(cts) == pagesize
		if isfull {
			cts = cts[:pagesize-1]
		}
		if !isleft {
			libs.Reverse(cts)
		}
		CTGetLikes(cts, user_id)
		return cts, isfull, nil
	} else {
		perms, err := USPermissions(user_id)
		if err != nil {
			return nil, false, err
		}

		var ids []int
		perm_str := libs.JoinArray(perms)
		if isleft {
			ids, err = libs.DBSelectInts(fmt.Sprintf("select distinct contest_id from contest_permissions where contest_id<=%d and permission_id in (%s) order by contest_id desc limit %d", bound, perm_str, pagesize))
		} else {
			ids, err = libs.DBSelectInts(fmt.Sprintf("select distinct contest_id from contest_permissions where contest_id>=%d and permission_id in (%s) order by contest_id limit %d", bound, perm_str, pagesize))
		}
		if err != nil {
			return nil, false, err
		}

		isfull := len(ids) == pagesize
		if isfull {
			ids = ids[:pagesize-1]
		}
		id_str := libs.JoinArray(ids)
		if len(ids) > 0 {
			err = libs.DBSelectAll(&cts, "select * from contests where contest_id in (" + id_str + ")")
			if err != nil {
				return nil, false, err
			}
		}
		sort.Slice(cts, func(i, j int) bool { return cts[i].Id > cts[j].Id })

		CTGetLikes(cts, user_id)
		if user_id > 0 && len(ids) > 0 {
			for i := range cts {
				cts[i].RegisterStatus = 1
			}
			ids, err = libs.DBSelectInts("select contest_id from contest_permissions where contest_id in ("+id_str+") and permission_id=?", -user_id)
			if err != nil {
				return nil, false, err
			}
			sort.Ints(ids)
			for i := range cts {
				if libs.HasInt(ids, cts[i].Id) {
					cts[i].RegisterStatus = 0
				}
			}
			ids, err = libs.DBSelectInts("select contest_id from contest_participants where contest_id in ("+id_str+") and user_id=?", user_id)
			if err != nil {
				return nil, false, err
			}
			sort.Ints(ids)
			for i := range cts {
				if libs.HasInt(ids, cts[i].Id) {
					cts[i].RegisterStatus = 2
				}
				if cts[i].RegisterStatus == 1 && cts[i].EndTime.Before(time.Now()) {
					cts[i].RegisterStatus = 0
				} else if cts[i].RegisterStatus == 2 && cts[i].StartTime.Before(time.Now()) {
					cts[i].RegisterStatus = 0
				}
			}
		}
		return cts, isfull, nil
	}
}

func CTGetLikes(cts []Contest, user_id int) {
	if user_id < 0 {
		return
	}
	ids := []int{}
	for _, i := range cts {
		ids = append(ids, i.Id)
	}
	ret := GetLikes(CONTEST, user_id, ids)
	for i := range cts {
		cts[i].Liked = libs.HasInt(ret, cts[i].Id)
	}
}

func CTQuery(contest_id, user_id int) (Contest, error) {
	var contest Contest
	err := libs.DBSelectSingle(&contest, "select * from contests where contest_id=?", contest_id)
	if err != nil {
		return contest, err
	}
	contest.Liked = GetLike(CONTEST, user_id, contest_id)
	return contest, err
}

func CTGetProblems(contest_id int) ([]Problem, error) {
	var problems []Problem
	err := libs.DBSelectAll(&problems, "select a.problem_id, title from ((select problem_id from contest_problems where contest_id=?) as a join problems on a.problem_id=problems.problem_id)", contest_id)
	return problems, err
}

func CTCreate() (int64, error) {
	start := time.Now().AddDate(0, 0, 1)
	return libs.DBInsertGetId("insert into contests values (null, \"New Contest\", ?, ?, 0, 0, 0, 0, 0)", start, start.Add(time.Hour))
}

func CTModify(contest_id int, title string, start time.Time, last int, pretest int, score_private int) error {
	_, err := libs.DBUpdate("update contests set title=?, start_time=?, end_time=?, pretest=?, score_private=? where contest_id=?", title, start, start.Add(time.Duration(last)*time.Minute), pretest, score_private, contest_id)
	return err
}

func CTGetPermissions(contest_id int) ([]Permission, error) {
	var p []Permission
	err := libs.DBSelectAll(&p, "select a.permission_id, permission_name from ((select permission_id from contest_permissions where contest_id=? and permission_id>0) as a join permissions on a.permission_id=permissions.permission_id)", contest_id)
	return p, err
}

func CTGetManagers(contest_id int) ([]User, error) {
	var u []User
	err := libs.DBSelectAll(&u, "select user_id, user_name from ((select permission_id from contest_permissions where contest_id=? and permission_id<0) as a join user_info on -permission_id=user_id)", contest_id)
	return u, err
}

func CTGetParticipants(contest_id int) ([]User, error) {
	var u []User
	err := libs.DBSelectAll(&u, "select a.user_id, user_name, rating from ((select user_id from contest_participants where contest_id=?) as a join user_info on a.user_id=user_info.user_id)", contest_id)
	return u, err
}

func CTAddPermission(contest_id, permission_id int) error {
	_, err := libs.DBUpdate("insert ignore into contest_permissions values (?, ?)", contest_id, permission_id)
	return err
}

func CTDeletePermission(contest_id, permission_id int) error {
	_, err := libs.DBUpdate("delete from contest_permissions where contest_id=? and permission_id=?", contest_id, permission_id)
	return err
}

func CTAddProblem(contest_id, problem_id int) error {
	_, err := libs.DBUpdate("insert ignore into contest_problems values (?, ?)", contest_id, problem_id)
	return err
}

func CTDeleteProblem(contest_id, problem_id int) error {
	_, err := libs.DBUpdate("delete from contest_problems where contest_id=? and problem_id=?", contest_id, problem_id)
	return err
}

func CTAddParticipant(contest_id, user_id int) error {
	affect, err := libs.DBUpdateGetAffected("insert ignore into contest_participants values (?, ?)", contest_id, user_id)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("update contests set registrants = registrants + ? where contest_id=?", affect, contest_id)
	return err
}

func CTDeleteParticipant(contest_id, user_id int) error {
	affect, err := libs.DBUpdateGetAffected("delete from contest_participants where contest_id=? and user_id=?", contest_id, user_id)
	if err != nil {
		return err
	}
	_, err = libs.DBUpdate("update contests set registrants = registrants - ? where contest_id=?", affect, contest_id)
	return err
}

func CTRegistered(contest_id, user_id int) bool {
	count, err := libs.DBSelectSingleInt("select count(*) from contest_participants where contest_id=? and user_id=?", contest_id, user_id)
	return err == nil && count > 0
}

func CTHasProblem(contest_id, problem_id int) bool {
	count, err := libs.DBSelectSingleInt("select count(*) from contest_problems where contest_id=? and problem_id=?", contest_id, problem_id)
	return err == nil && count > 0
}

func CTExists(contest_id int) bool {
	count, err := libs.DBSelectSingleInt("select count(*) from contests where contest_id=?", contest_id)
	return err == nil && count > 0
}

func CTPretestOnly(contest_id int) bool {
	pretest, err := libs.DBSelectSingleInt("select pretest from contests where contest_id=?", contest_id)
	return err == nil || pretest > 0
}