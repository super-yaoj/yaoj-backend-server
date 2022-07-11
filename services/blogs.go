package services

import (
	"fmt"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func BLCanSee(ctx *gin.Context, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author, private from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return ISAdmin(ctx) || GetUserId(ctx) == blog.Author || !blog.Private
	}
}

func BLCanEdit(ctx *gin.Context, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return ISAdmin(ctx) || GetUserId(ctx) == blog.Author
	}
}

type BlogCreateParam struct {
	UserID  int    `session:"user_id"`
	Private *int   `body:"private"`
	Title   string `body:"title"`
	Content string `body:"content"`
}

func BlogCreate(ctx *gin.Context, param BlogCreateParam) {
	if param.Private == nil {
		return
	}
	if !internal.BLValidTitle(param.Title) {
		libs.APIWriteBack(ctx, 400, "invalid title", nil)
		return
	}
	id, err := internal.BLCreate(param.UserID, *param.Private, param.Title, param.Content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
	}
}

type BlogEditParam struct {
	BlogID *int `body:"blog_id"`
}

func BlogEdit(ctx *gin.Context, param BlogEditParam) {
	if param.BlogID == nil {
		return
	}
	if !internal.BLExists(*param.BlogID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	private, ok := libs.PostIntRange(ctx, "private", 0, 1)
	if !ok {
		return
	}
	if !BLCanEdit(ctx, *param.BlogID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	title := ctx.PostForm("title")
	if !internal.BLValidTitle(title) {
		libs.APIWriteBack(ctx, 400, "invalid title", nil)
		return
	}
	content := ctx.PostForm("content")
	err := internal.BLEdit(*param.BlogID, private, title, content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type BlogDelParam struct {
	BlogID int `query:"blog_id" binding:"required"`
}

func BlogDel(ctx *gin.Context, param BlogDelParam) {
	if !BLCanEdit(ctx, param.BlogID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.BLDelete(param.BlogID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type BlogGetParam struct {
	BlogID int `query:"blog_id" binding:"required"`
	UserID int `session:"user_id"`
}

func BlogGet(ctx *gin.Context, param BlogGetParam) {
	if !internal.BLExists(param.BlogID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !BLCanSee(ctx, param.BlogID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	blog, err := internal.BLQuery(param.BlogID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		ret, _ := libs.Struct2Map(blog)
		libs.APIWriteBack(ctx, 200, "", ret)
	}
}

type BlogListParam struct {
	UserID   *int `query:"user_id"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize *int `query:"pagesize"`
}

func BlogList(ctx *gin.Context, param BlogListParam) {
	if param.UserID != nil {
		blogs, err := internal.BLListUser(*param.UserID, GetUserId(ctx))
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": blogs})
		}
	} else {
		if param.PageSize == nil || *param.PageSize > 100 || *param.PageSize < 1 {
			libs.APIWriteBack(ctx, 400, fmt.Sprintf("invalid request: parameter pagesize should be in [%d, %d]", 1, 100), nil)
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
		blogs, isfull, err := internal.BLListAll(bound, *param.PageSize, GetUserId(ctx), param.Left != nil, ISAdmin(ctx))
		if err != nil {
			libs.APIInternalError(ctx, err)
			return
		}
		libs.APIWriteBack(ctx, 200, "", map[string]any{"isfull": isfull, "data": blogs})
	}
}
