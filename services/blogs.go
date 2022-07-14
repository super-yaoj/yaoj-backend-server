package services

import (
	"fmt"
	"yao/internal"
	"yao/libs"
)

// 1. blog exist
// 2. user registered
// 3. admin or author or blog is public
func BLCanSee(auth Auth, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author, private from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else if auth.UserID == 0 { // unregistered
		return false
	} else {
		return libs.IsAdmin(auth.UserGrp) || auth.UserID == blog.Author || !blog.Private
	}
}

func BLCanEdit(auth Auth, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return libs.IsAdmin(auth.UserGrp) || auth.UserID == blog.Author
	}
}

type BlogCreateParam struct {
	UserID  int    `session:"user_id" validate:"gte=0"`
	Private int    `body:"private" binding:"required"`
	Title   string `body:"title"`
	Content string `body:"content"`
}

func BlogCreate(ctx Context, param BlogCreateParam) {
	if !internal.BLValidTitle(param.Title) {
		ctx.JSONAPI(400, "invalid title", nil)
		return
	}
	id, err := internal.BLCreate(param.UserID, param.Private, param.Title, param.Content)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"id": id})
	}
}

type BlogEditParam struct {
	BlogID  int    `body:"blog_id" binding:"required"`
	Private int    `body:"private" binding:"required" validate:"gte=0,lte=1"`
	Title   string `body:"title"`
	Content string `body:"content"`
	Auth
}

func BlogEdit(ctx Context, param BlogEditParam) {
	if !internal.BLExists(param.BlogID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !BLCanEdit(param.Auth, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	if !internal.BLValidTitle(param.Title) {
		ctx.JSONAPI(400, "invalid title", nil)
		return
	}
	err := internal.BLEdit(param.BlogID, param.Private, param.Title, param.Content)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type BlogDelParam struct {
	BlogID int `query:"blog_id" binding:"required"`
	Auth
}

func BlogDel(ctx Context, param BlogDelParam) {
	if !BLCanEdit(param.Auth, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.BLDelete(param.BlogID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type BlogGetParam struct {
	BlogID int `query:"blog_id" binding:"required"`
	Auth
}

func BlogGet(ctx Context, param BlogGetParam) {
	if !internal.BLExists(param.BlogID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !BLCanSee(param.Auth, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	blog, err := internal.BLQuery(param.BlogID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ret, _ := libs.Struct2Map(blog)
		ctx.JSONAPI(200, "", ret)
	}
}

type BlogListParam struct {
	BlogUserID *int `query:"user_id"`
	Left       *int `query:"left"`
	Right      *int `query:"right"`
	PageSize   *int `query:"pagesize"`
	Auth
}

func BlogList(ctx Context, param BlogListParam) {
	if param.BlogUserID != nil {
		blogs, err := internal.BLListUser(*param.BlogUserID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(200, "", map[string]any{"data": blogs})
		}
	} else {
		if param.PageSize == nil || *param.PageSize > 100 || *param.PageSize < 1 {
			ctx.JSONAPI(400, fmt.Sprintf("invalid request: parameter pagesize should be in [%d, %d]", 1, 100), nil)
			return
		}
		var bound int
		if param.Left != nil {
			bound = *param.Left
		} else if param.Right != nil {
			bound = *param.Right
		} else {
			return
		}
		blogs, isfull, err := internal.BLListAll(
			bound, *param.PageSize, param.UserID, param.Left != nil, libs.IsAdmin(param.UserGrp),
		)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
		ctx.JSONAPI(200, "", map[string]any{"isfull": isfull, "data": blogs})
	}
}
