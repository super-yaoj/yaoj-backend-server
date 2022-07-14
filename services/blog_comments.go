package services

import (
	"yao/internal"
	"yao/libs"
)

type BlogCmntCreateParam struct {
	BlogID int `body:"blog_id" binding:"required"`
	Auth
}

func BlogCmntCreate(ctx Context, param BlogCmntCreateParam) {
	if param.UserID <= 0 {
		ctx.JSONAPI(401, "", nil)
		return
	}
	if !BLCanSee(param.UserID, param.UserGrp, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	content := ctx.PostForm("content")
	id, err := internal.BLCreateComment(param.BlogID, param.UserID, content)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"id": id})
	}
}

type BlogCmntGetParam struct {
	BlogID int `query:"blog_id" binding:"required"`
	Auth
}

func BlogCmntGet(ctx Context, param BlogCmntGetParam) {
	if !internal.BLExists(param.BlogID) {
		ctx.JSONAPI(404, "", nil)
		return
	}
	if !BLCanSee(param.UserID, param.UserGrp, param.BlogID) {
		ctx.JSONAPI(403, "", nil)
		return
	}
	comments, err := internal.BLGetComments(param.BlogID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", map[string]any{"data": comments})
	}
}

type BlogCmntDelParam struct {
	CmntID int `query:"comment_id" binding:"required"`
	Auth
}

func BlogCmntDel(ctx Context, param BlogCmntDelParam) {
	var comment internal.Comment
	err := libs.DBSelectSingle(&comment, "select author, blog_id from blog_comments where comment_id=?", param.CmntID)
	if err != nil {
		ctx.JSONAPI(404, "", nil)
	} else if !libs.IsAdmin(param.UserGrp) && comment.Author != param.UserID {
		ctx.JSONAPI(403, "", nil)
	} else {
		err = internal.BLDeleteComment(param.CmntID, comment.BlogId)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}
}
