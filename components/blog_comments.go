package components

import (
	"yao/controllers"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func BLCreateComment(ctx *gin.Context) {
	user_id := GetUserId(ctx)
	if user_id < 0 {
		libs.APIWriteBack(ctx, 401, "", nil)
		return
	}
	blog_id, ok := libs.PostInt(ctx, "blog_id")
	if !ok {
		return
	}
	if !BLCanSee(ctx, blog_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	content := ctx.PostForm("content")
	id, err := controllers.BLCreateComment(blog_id, user_id, content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
	}
}

func BLGetComments(ctx *gin.Context) {
	blog_id, ok := libs.GetInt(ctx, "blog_id")
	if !ok {
		return
	}
	if !controllers.BLExists(blog_id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !BLCanSee(ctx, blog_id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	comments, err := controllers.BLGetComments(blog_id, GetUserId(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"data": comments})
	}
}

func BLDeleteComment(ctx *gin.Context) {
	id, ok := libs.GetInt(ctx, "comment_id")
	if !ok {
		return
	}
	user_id := GetUserId(ctx)
	var comment controllers.Comment
	err := libs.DBSelectSingle(&comment, "select author, blog_id from blog_comments where comment_id=?", id)
	if err != nil {
		libs.APIWriteBack(ctx, 404, "", nil)
	} else if !ISAdmin(ctx) && comment.Author != user_id {
		libs.APIWriteBack(ctx, 403, "", nil)
	} else {
		err = controllers.BLDeleteComment(id, comment.BlogId)
		if err != nil {
			libs.APIInternalError(ctx, err)
		}
	}
}
