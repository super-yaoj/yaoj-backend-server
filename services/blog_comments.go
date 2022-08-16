package services

import (
	"net/http"
	"yao/internal"
)

type BlogCmntCreateParam struct {
	Auth
	BlogID int `body:"blog_id" validate:"required,blogid"`
}

func BlogCmntCreate(ctx *Context, param BlogCmntCreateParam) {
	param.NewPermit().TrySeeBlog(param.BlogID).Success(func(any) {
		content := ctx.PostForm("content")
		id, err := internal.BlogCreateComment(param.BlogID, param.UserID, content)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogCmntGetParam struct {
	Auth
	BlogID int `query:"blog_id" validate:"required,blogid"`
}

func BlogCmntGet(ctx *Context, param BlogCmntGetParam) {
	param.NewPermit().TrySeeBlog(param.BlogID).Success(func(any) {
		comments, err := internal.BlogGetComments(param.BlogID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": comments})
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogCmntDelParam struct {
	Auth
	CmntID int `query:"comment_id" validate:"required,cmntid"`
}

func BlogCmntDel(ctx *Context, param BlogCmntDelParam) {
	param.NewPermit().TryEditBlogCmnt(param.CmntID).Success(func(any) {
		err := internal.BlogDeleteComment(param.CmntID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}
