package services

import (
	"fmt"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type AnceCreateParam struct {
	BlogID   *int `body:"blog_id"`
	Priority *int `body:"priority"`
}

func AnceCreate(ctx *gin.Context, param AnceCreateParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if param.BlogID == nil || param.Priority == nil {
		return
	}
	if *param.Priority < 1 || *param.Priority > 10 {
		libs.APIWriteBack(ctx, 400, fmt.Sprintf("invalid request: parameter priority should be in [%d, %d]", 1, 10), nil)
	}

	count, err := libs.DBSelectSingleInt("select count(*) from blogs where blog_id=?", *param.BlogID)
	if err != nil || count == 0 {
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	err = internal.ANCreate(*param.BlogID, *param.Priority)
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
}

type AnceGetParam struct {
	UserID int `session:"user_id"`
}

func AnceGet(ctx *gin.Context, param AnceGetParam) {
	libs.APIWriteBack(ctx, 200, "", map[string]any{"data": internal.ANQuery(param.UserID)})
}

type AnceDelParam struct {
	ID int `query:"id" binding:"required"`
}

func AnceDel(ctx *gin.Context, param AnceDelParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	internal.ANDelete(param.ID)
}
