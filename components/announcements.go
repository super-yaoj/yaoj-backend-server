package components

import (
	"yao/controllers"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func ANCreate(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	blog_id, ok := libs.PostInt(ctx, "blog_id")
	priority, ok1 := libs.PostIntRange(ctx, "priority", 1, 10)
	if !ok || !ok1 {
		return
	}

	count, err := libs.DBSelectSingleInt("select count(*) from blogs where blog_id=?", blog_id)
	if err != nil || count == 0 {
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	err = controllers.ANCreate(blog_id, priority)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
}

func ANQuery(ctx *gin.Context) {
	libs.APIWriteBack(ctx, 200, "", map[string]any{"data": controllers.ANQuery(GetUserId(ctx))})
}

func ANDelete(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, ok := libs.GetInt(ctx, "id")
	if !ok {
		return
	}
	controllers.ANDelete(id)
}
