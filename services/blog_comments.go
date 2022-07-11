package services

import (
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type BlogCmntCreateParam struct {
	UserID int `session:"user_id"`
	BlogID int `body:"blog_id" binding:"required"`
}

func BlogCmntCreate(ctx *gin.Context, param BlogCmntCreateParam) {
	if param.UserID <= 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	if !BLCanSee(ctx, param.BlogID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	content := ctx.PostForm("content")
	id, err := internal.BLCreateComment(param.BlogID, param.UserID, content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
	}
}

type BlogCmntGetParam struct {
	BlogID int `query:"blog_id" binding:"required"`
}

func BlogCmntGet(ctx *gin.Context, param BlogCmntGetParam) {
	if !internal.BLExists(param.BlogID) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !BLCanSee(ctx, param.BlogID) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	comments, err := internal.BLGetComments(param.BlogID, GetUserId(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": comments})
	}
}

type BlogCmntDelParam struct {
	CmntID int `query:"comment_id" binding:"required"`
	UserID int `session:"user_id"`
}

func BlogCmntDel(ctx *gin.Context, param BlogCmntDelParam) {
	var comment internal.Comment
	err := libs.DBSelectSingle(&comment, "select author, blog_id from blog_comments where comment_id=?", param.CmntID)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
	} else if !ISAdmin(ctx) && comment.Author != param.UserID {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		err = internal.BLDeleteComment(param.CmntID, comment.BlogId)
		if err != nil {
			libs.APIInternalError(ctx, err)
		}
	}
}
