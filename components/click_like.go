package components

import (
	"yao/controllers"
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
		if !controllers.BLExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = controllers.ClickLike(controllers.BLOG, id, user_id)
	case "comment":
		if !controllers.BLCommentExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = controllers.ClickLike(controllers.COMMENT, id, user_id)
	case "problem":
		if !controllers.PRExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = controllers.ClickLike(controllers.PROBLEM, id, user_id)
	case "contest":
		if !controllers.CTExists(id) {
			libs.APIWriteBack(ctx, 404, "", nil)
			return
		}
		err = controllers.ClickLike(controllers.CONTEST, id, user_id)
	default:
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}
