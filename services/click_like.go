package services

import (
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func ClickLike(ctx *gin.Context) {
	target := ctx.PostForm("target")
	user_id := GetUserId(ctx)
	if user_id < 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	id, ok := libs.PostInt(ctx, "id")
	if !ok {
		return
	}
	var err error
	switch target {
	case "blog":
		if !internal.BLExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.BLOG, id, user_id)
	case "comment":
		if !internal.BLCommentExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.COMMENT, id, user_id)
	case "problem":
		if !internal.PRExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.PROBLEM, id, user_id)
	case "contest":
		if !internal.CTExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = internal.ClickLike(internal.CONTEST, id, user_id)
	default:
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}
