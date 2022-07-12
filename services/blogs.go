package services

import (
	"fmt"
	"yao/internal"
	"yao/libs"
)

func BLCanSee(user_id, user_group, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author, private from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return libs.IsAdmin(user_group) || user_id == blog.Author || !blog.Private
	}
}

func BLCanEdit(user_id, user_group, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return libs.IsAdmin(user_group) || user_id == blog.Author
	}
}

type BlogCreateParam struct {
	UserID  int    `session:"user_id"`
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
	UserID  int    `session:"user_id"`
	UserGrp int    `session:"user_group"`
}

func BlogEdit(ctx Context, param BlogEditParam) {
	if !internal.BLExists(param.BlogID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !BLCanEdit(param.UserID, param.UserGrp, param.BlogID) {
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
	BlogID  int `query:"blog_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func BlogDel(ctx Context, param BlogDelParam) {
	if !BLCanEdit(param.UserID, param.UserGrp, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	err := internal.BLDelete(param.BlogID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type BlogGetParam struct {
	BlogID  int `query:"blog_id" binding:"required"`
	UserID  int `session:"user_id"`
	UserGrp int `session:"user_group"`
}

func BlogGet(ctx Context, param BlogGetParam) {
	if !internal.BLExists(param.BlogID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !BLCanSee(param.UserID, param.UserGrp, param.BlogID) {
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
	UserID    *int `query:"user_id"`
	Left      *int `query:"left"`
	Right     *int `query:"right"`
	PageSize  *int `query:"pagesize"`
	CurUserID int  `session:"user_id"`
	UserGrp   int  `session:"user_group"`
}

func BlogList(ctx Context, param BlogListParam) {
	if param.UserID != nil {
		blogs, err := internal.BLListUser(*param.UserID, param.CurUserID)
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
			bound, *param.PageSize, param.CurUserID, param.Left != nil, libs.IsAdmin(param.UserGrp),
		)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
		ctx.JSONAPI(200, "", map[string]any{"isfull": isfull, "data": blogs})
	}
}
