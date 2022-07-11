package services

import (
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type ClickLikeParam struct {
	Target string `body:"target"`
	UserID int    `session:"user_id"`
}

func ClickLike(ctx *gin.Context, param ClickLikeParam) {
	if param.UserID < 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	id, ok := libs.PostInt(ctx, "id")
	if !ok {
		return
	}
	var err error
	switch param.Target {
	case "blog":
		if !internal.BLExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.BLOG, id, param.UserID)
	case "comment":
		if !internal.BLCommentExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.COMMENT, id, param.UserID)
	case "problem":
		if !internal.PRExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.PROBLEM, id, param.UserID)
	case "contest":
		if !internal.CTExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.CONTEST, id, param.UserID)
	default:
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}
