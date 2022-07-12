package services

import (
	"fmt"
	"yao/internal"
	"yao/libs"
)

type AnceCreateParam struct {
	BlogID   int `body:"blog_id" binding:"required"`
	Priority int `body:"priority" binding:"required"`
	UserGrp  int `session:"user_group" validate:"admin"`
}

func AnceCreate(ctx Context, param AnceCreateParam) {
	if param.Priority < 1 || param.Priority > 10 {
		ctx.JSONAPI(400, fmt.Sprintf("invalid request: parameter priority should be in [%d, %d]", 1, 10), nil)
	}

	count, err := libs.DBSelectSingleInt("select count(*) from blogs where blog_id=?", param.BlogID)
	if err != nil || count == 0 {
		ctx.JSONAPI(400, "invalid request", nil)
		return
	}
	err = internal.ANCreate(param.BlogID, param.Priority)
	if err != nil {
		ctx.ErrorAPI(err)
		return
	}
}

type AnceGetParam struct {
	UserID int `session:"user_id"`
}

func AnceGet(ctx Context, param AnceGetParam) {
	ctx.JSONAPI(200, "", map[string]any{"data": internal.ANQuery(param.UserID)})
}

type AnceDelParam struct {
	ID      int `query:"id" binding:"required"`
	UserGrp int `session:"user_group" validate:"admin"`
}

func AnceDel(ctx Context, param AnceDelParam) {
	internal.ANDelete(param.ID)
}
