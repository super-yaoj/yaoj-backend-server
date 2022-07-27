package services

import (
	"time"
	"yao/internal"
	"yao/libs"
)

// pagination query param
type Page struct {
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
}

func (r *Page) CanBound() bool {
	return r.Left != nil || r.Right != nil
}

func (r *Page) Bound() int {
	if r.Left != nil {
		return *r.Left
	} else if r.Right != nil {
		return *r.Right
	}
	return 0
}
func (r *Page) IsLeft() bool {
	return r.Left != nil
}

// authorization stored in session
type Auth struct {
	UserID  int `session:"user_id" validate:"gte=0"`
	UserGrp int `session:"user_group" validate:"gte=0,lte=3"`

	// if from contest
	ctstOrigin int
}

func (auth *Auth) IsAdmin() bool {
	return libs.IsAdmin(auth.UserGrp)
}

func (r *Auth) SetCtst(ctstid int) *Auth {
	r.ctstOrigin = ctstid
	return r
}

// 1. valid user_id
// 2. admin or problem permission
func (auth *Auth) CanEditProb(problem_id int) bool {
	if auth.IsAdmin() {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, -auth.UserID)
	return count > 0
}
func (auth *Auth) CanSeeProb(problem_id int) bool {
	// public problem
	if auth.UserID == 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, libs.DefaultGroup)
		return count > 0
	}
	// can edit
	if auth.CanEditProb(problem_id) {
		return true
	}
	// permitted
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select * from problem_permissions where problem_id=?) as a join (select * from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", problem_id, auth.UserID)
	return count > 0
}

func (auth *Auth) CanSeeProbInCtst(problem_id, contest_id int) bool {
	contest, _ := internal.CTQuery(contest_id, auth.UserID)
	if auth.CanEnterCtst(contest) &&
		contest.StartTime.Before(time.Now()) && contest.EndTime.After(time.Now()) {
		return internal.CTHasProblem(contest_id, problem_id)
	}
	return false
}

func (auth *Auth) CanEditCtst(contest_id int) bool {
	if auth.IsAdmin() {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -auth.UserID)
	return count > 0
}

func (auth *Auth) CanSeeCtst(contest_id int) bool {
	if auth.UserID == 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, libs.DefaultGroup)
		return count > 0
	}
	if auth.CanEditCtst(contest_id) {
		return true
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select permission_id from contest_permissions where contest_id=?) as a join (select permission_id from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", contest_id, auth.UserID)
	return count > 0
}

func (auth *Auth) CanEnterCtst(contest internal.Contest) bool {
	if contest.StartTime.After(time.Now()) {
		return false
	} else if contest.EndTime.After(time.Now()) {
		return internal.CTRegistered(contest.Id, auth.UserID)
	} else {
		return auth.CanSeeCtst(contest.Id)
	}
}

func (auth *Auth) CanTakeCtst(contest internal.Contest) bool {
	if contest.EndTime.After(time.Now()) {
		return !auth.CanEditCtst(contest.Id) && auth.CanSeeCtst(contest.Id)
	}
	return false
}

// 许可证
type Permit[DataType any] struct {
	data DataType
	// accepted or not accepted
	status bool
}

func (r *Permit[T]) Then(callback func(a T)) *Permit[T] {
	if r.status {
		callback(r.data)
	}
	return r
}
func (r *Permit[T]) Else(callback func(a T)) *Permit[T] {
	if !r.status {
		callback(r.data)
	}
	return r
}

// if can't see -> unaccepted
// if must see from contest, data represents contest id (none zero),
// otherwise zero.
func (auth *Auth) TrySeeProb(probid int) *Permit[int] {
	if auth.CanSeeProb(probid) {
		return &Permit[int]{status: true, data: 0}
	}
	if auth.ctstOrigin > 0 && auth.CanSeeProbInCtst(probid, auth.ctstOrigin) {
		return &Permit[int]{status: true, data: auth.ctstOrigin}
	}
	return &Permit[int]{status: false}
}
