package service

import (
	"net/http"
	"time"
	"yao/config"
	"yao/db"
	"yao/internal"

	"github.com/super-yaoj/yaoj-utils/promise"
)

// authorization stored in session
type Auth struct {
	UserID  int `session:"user_id" validate:"gte=0"`
	UserGrp int `session:"user_group" validate:"gte=0,lte=3"`
}

func (auth *Auth) IsAdmin() bool {
	return internal.IsAdmin(auth.UserGrp)
}

// 1. valid user_id
// 2. admin or problem permission
func (auth *Auth) CanEditProb(problem_id int) bool {
	if auth.IsAdmin() {
		return true
	}
	count, _ := db.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, -auth.UserID)
	return count > 0
}
func (auth *Auth) CanSeeProb(problem_id int) bool {
	// public problem
	if auth.UserID == 0 {
		count, _ := db.DBSelectSingleInt("select count(*) from problem_permissions where problem_id=? and permission_id=?", problem_id, config.Global.DefaultGroup)
		return count > 0
	}
	// can edit
	if auth.CanEditProb(problem_id) {
		return true
	}
	// permitted
	count, _ := db.DBSelectSingleInt("select count(*) from ((select * from problem_permissions where problem_id=?) as a join (select * from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", problem_id, auth.UserID)
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
	count, _ := db.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, -auth.UserID)
	return count > 0
}

func (auth *Auth) CanSeeCtst(contest_id int, can_edit bool) bool {
	if can_edit {
		return true
	}
	if auth.UserID == 0 {
		count, _ := db.DBSelectSingleInt("select count(*) from contest_permissions where contest_id=? and permission_id=?", contest_id, config.Global.DefaultGroup)
		return count > 0
	}
	count, _ := db.DBSelectSingleInt("select count(*) from ((select permission_id from contest_permissions where contest_id=?) as a join (select permission_id from user_permissions where user_id=?) as b on a.permission_id=b.permission_id)", contest_id, auth.UserID)
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

func (auth *Auth) CanEditSubm(sub internal.SubmissionBase) bool {
	return auth.CanEditProb(sub.ProblemId) ||
		(sub.ContestId > 0 && auth.CanEditCtst(sub.ContestId))
}

// 许可证
type Permit struct {
	*Auth
	*promise.SyncPromise[any, bool]
}

func (auth *Auth) NewPermit() *Permit {
	return &Permit{
		auth,
		promise.NewSyncPromise(
			func(p bool) bool { return !p },
			func() (any, bool) { return nil, true },
		),
	}
}

func (p *Permit) Try(callback func() (any, bool)) *Permit {
	p.SyncPromise.Then(func(t any) (any, bool) { return callback() })
	return p
}

func (p *Permit) Then(callback func(any) (any, bool)) *Permit {
	p.SyncPromise.Then(callback)
	return p
}

func (p *Permit) Success(callback func(any)) *Permit {
	p.SyncPromise.Then(func(t any) (any, bool) {
		callback(t)
		return t, true
	})
	return p
}

func (p *Permit) Fail(callback func()) {
	p.SyncPromise.Catch(func(b bool) {
		callback()
	})
}

func (p *Permit) FailAPIStatusForbidden(ctx Context) {
	p.Fail(func()  {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
	})
}
func (p *Permit) FailRPCStatusForbidden(ctx Context) {
	p.Fail(func() {
		ctx.JSONRPC(http.StatusForbidden, -32600, "", nil)
	})
}

func (p *Permit) AsAdmin() *Permit {
	return p.Try(func() (any, bool) {
		return nil, p.IsAdmin()
	})
}

//user registered and user group is at least normal user
func (p *Permit) AsNormalUser() *Permit {
	return p.Try(func() (any, bool) {
		return nil, p.UserID > 0 && !internal.IsBanned(p.UserGrp)
	})
}

//ctstid=0 means no contest
// if can't see -> unaccepted
// if must see from contest, data represents contest id (none zero),
// otherwise zero.
func (p *Permit) TrySeeProb(probid int, ctstid int) *Permit {
	return p.Try(func() (a any, b bool) {
		if p.CanSeeProb(probid) {
			return 0, true
		}
		if ctstid > 0 && p.CanSeeProbInCtst(probid, ctstid) {
			return ctstid, true
		}
		return 0, false
	})
}

func (p *Permit) TryEditProb(probid int) *Permit {
	return p.Try(func() (any, bool) {
		return nil, p.CanEditProb(probid)
	})
}

type PermitCtst struct {
	*internal.Contest
	CanEdit bool
}

func (p *Permit) TryEnterCtst(ctstid int) *Permit {
	return p.Try(func() (any, bool) {
		ctst, _ := internal.CTQuery(ctstid, p.UserID)
		can_edit := p.CanEditCtst(ctstid)
		return PermitCtst{&ctst, can_edit}, p.CanEnterCtst(ctst, can_edit)
	})
}

func (p *Permit) TryEditCtst(ctstid int) *Permit {
	return p.Try(func() (any, bool) {
		return nil, p.CanEditCtst(ctstid)
	})
}

func (p *Permit) TryTakeCtst(ctstid int) *Permit {
	return p.Try(func() (any, bool) {
		ctst, _ := internal.CTQuery(ctstid, p.UserID)
		can_edit := p.CanEditCtst(ctstid)
		return PermitCtst{&ctst, can_edit}, p.CanTakeCtst(ctst, can_edit)
	})
}

func (p *Permit) TrySeeCtst(ctstid int) *Permit {
	return p.Try(func() (any, bool) {
		can_edit := p.CanEditCtst(ctstid)
		return can_edit, p.CanSeeCtst(ctstid, can_edit)
	})
}

func (p *Permit) TrySeeBlog(blogid int) *Permit {
	return p.Try(func() (any, bool) {
		var blog internal.Blog
		db.DBSelectSingle(&blog, "select blog_id, author, private from blogs where blog_id=?", blog)
		return nil, p.IsAdmin() || p.UserID == blog.Author || !blog.Private
	})
}

func (p *Permit) TryEditBlog(blogid int) *Permit {
	return p.Try(func() (any, bool) {
		var blog internal.Blog
		db.DBSelectSingle(&blog, "select blog_id, author from blogs where blog_id=?", blogid)
		return nil, p.IsAdmin() || p.UserID == blog.Author
	})
}

func (p *Permit) TryEditBlogCmnt(cmntid int) *Permit {
	return p.Try(func() (any, bool) {
		var comment internal.Comment
		db.DBSelectSingle(&comment, "select author, blog_id from blog_comments where comment_id=?", cmntid)
		return struct{}{}, p.IsAdmin() || comment.Author == p.UserID
	})
}

type PermitSubm struct {
	internal.Submission
	ByProb  bool
	CanEdit bool
}

func (p *Permit) TrySeeSubm(submid int) *Permit {
	ret, _ := internal.SMQuery(submid)
	return p.TrySeeProb(ret.ProblemId, ret.ContestId).Then(func(a any) (any, bool) {
		by_problem := a.(int) > 0
		can_edit := p.CanEditSubm(ret.SubmissionBase)
		if !can_edit && ret.Submitter != p.UserID && !by_problem {
			return nil, false
		}
		return PermitSubm{ret, by_problem, can_edit}, true
	})
}

func (p *Permit) TryEditSubm(submid int) *Permit {
	return p.Try(func() (any, bool) {
		ret, _ := internal.SMGetBaseInfo(submid)
		return ret, p.CanEditSubm(ret)
	})
}