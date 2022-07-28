package services

import (
	"yao/internal"
)

type AnceCreateParam struct {
	BlogID   int `body:"blog_id" binding:"required" validate:"blogid"`
	Priority int `body:"priority" binding:"required" validate:"gte=1,lte=10"`
	UserGrp  int `session:"user_group" validate:"admin"`
}

func AnceCreate(ctx Context, param AnceCreateParam) {
	err := internal.ANCreate(param.BlogID, param.Priority)
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
