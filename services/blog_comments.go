package services

import (
	"net/http"
	"yao/internal"
	"yao/libs"
)

type BlogCmntCreateParam struct {
	BlogID int `body:"blog_id" binding:"required" validate:"blogid"`
	Auth
}

func BlogCmntCreate(ctx Context, param BlogCmntCreateParam) {
	if !BLCanSee(param.Auth, param.BlogID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	content := ctx.PostForm("content")
	id, err := internal.BLCreateComment(param.BlogID, param.UserID, content)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
	}
}

type BlogCmntGetParam struct {
	BlogID int `query:"blog_id" binding:"required" validate:"blogid"`
	Auth
}

func BlogCmntGet(ctx Context, param BlogCmntGetParam) {
	if !BLCanSee(param.Auth, param.BlogID) {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
		return
	}
	comments, err := internal.BLGetComments(param.BlogID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": comments})
	}
}

type BlogCmntDelParam struct {
	CmntID int `query:"comment_id" binding:"required" validate:"blogid"`
	Auth
}

func BlogCmntDel(ctx Context, param BlogCmntDelParam) {
	var comment internal.Comment
	libs.DBSelectSingle(&comment, "select author, blog_id from blog_comments where comment_id=?", param.CmntID)
	if !param.IsAdmin() && comment.Author != param.UserID {
		ctx.JSONAPI(http.StatusForbidden, "", nil)
	} else {
		err := internal.BLDeleteComment(param.CmntID, comment.BlogId)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}
}
