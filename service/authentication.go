package service

import (
	"net/http"
	"time"
	"yao/internal"
	"yao/libs"
)

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
	if auth.CanEnterCtst(contest, auth.CanEditCtst(contest_id)) &&
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

func (auth *Auth) CanSeeCtst(contest_id int, can_edit bool) bool {
	if can_edit {
		return true
	}
	if auth.UserID == 0 {
		count, _ := libs.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, libs.DefaultGroup)
		return count > 0
	}
	count, _ := libs.DBSelectSingleInt("select count(*) from ((select permission_id from contest_permissions where contest_id=?) as a join (select permission_id from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", contest_id, auth.UserID)
	return count > 0
}

func (auth *Auth) CanEnterCtst(contest internal.Contest, can_edit bool) bool {
	if can_edit {
		return true
	}
	if contest.StartTime.After(time.Now()) {
		return false
	} else if contest.EndTime.After(time.Now()) {
		return internal.CTRegistered(contest.Id, auth.UserID)
	} else {
		return auth.CanSeeCtst(contest.Id, can_edit)
	}
}

func (auth *Auth) CanTakeCtst(contest internal.Contest, can_edit bool) bool {
	if contest.EndTime.After(time.Now()) {
		return !can_edit && auth.CanSeeCtst(contest.Id, can_edit)
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
func (r *Permit[T]) ElseAPIStatusForbidden(ctx Context) {
	if !r.status {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
	}
}
func (r *Permit[T]) ElseRPCStatusForbidden(ctx Context) {
	if !r.status {
		ctx.JSONRPC(http.StatusForbidden, -32600, "", nil)
	}
}

func (auth *Auth) AsAdmin() *Permit[struct{}] {
	if auth.IsAdmin() {
		return &Permit[struct{}]{status: true, data: struct{}{}}
	}
	return &Permit[struct{}]{status: false, data: struct{}{}}
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

func (auth *Auth) TryEditProb(probid int) *Permit[struct{}] {
	if auth.CanEditProb(probid) {
		return &Permit[struct{}]{status: true}
	}
	return &Permit[struct{}]{status: false}
}

type PermitCtst struct {
	*internal.Contest
	CanEdit bool
}
func (auth *Auth) TryEnterCtst(ctstid int) *Permit[PermitCtst] {
	ctst, _ := internal.CTQuery(ctstid, auth.UserID)
	can_edit := auth.CanEditCtst(ctstid)
	if auth.CanEnterCtst(ctst, can_edit) {
		return &Permit[PermitCtst]{status: true, data: PermitCtst{&ctst, can_edit}}
	}
	return &Permit[PermitCtst]{status: false, data: PermitCtst{&ctst, can_edit}}
}

func (auth *Auth) TryEditCtst(ctstid int) *Permit[struct{}] {
	if auth.CanEditCtst(ctstid) {
		return &Permit[struct{}]{status: true}
	}
	return &Permit[struct{}]{status: false}
}

func (auth *Auth) TryTakeCtst(ctstid int) *Permit[PermitCtst] {
	ctst, _ := internal.CTQuery(ctstid, auth.UserID)
	can_edit := auth.CanEditCtst(ctstid)
	if auth.CanTakeCtst(ctst, can_edit) {
		return &Permit[PermitCtst]{status: true, data: PermitCtst{&ctst, can_edit}}
	}
	return &Permit[PermitCtst]{status: false, data: PermitCtst{&ctst, can_edit}}
}

func (auth *Auth) TrySeeCtst(ctstid int) *Permit[bool] {
	can_edit := auth.CanEditCtst(ctstid)
	if auth.CanSeeCtst(ctstid, can_edit) {
		return &Permit[bool]{status: true, data: can_edit}
	}
	return &Permit[bool]{status: false, data: can_edit}
}