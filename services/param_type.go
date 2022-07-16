package services

import (
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
}

func (auth *Auth) IsAdmin() bool {
	return libs.IsAdmin(auth.UserGrp)
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
