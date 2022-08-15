package services

import (
	"net/http"
	"yao/internal"
)

type AnceCreateParam struct {
	Auth
	BlogID   int `body:"blog_id" validate:"required,blogid"`
	Priority int `body:"priority" validate:"required,gte=1,lte=10"`
}

func AnceCreate(ctx Context, param AnceCreateParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		err := internal.AnceCreate(param.BlogID, param.Priority)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
	}).FailAPIStatusForbidden(ctx)
}

type AnceGetParam struct {
	Auth
}

func AnceGet(ctx Context, param AnceGetParam) {
	ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": internal.AnceQuery(param.UserID)})
}

type AnceDelParam struct {
	Auth
	ID      int `query:"id" validate:"required"`
}

func AnceDel(ctx Context, param AnceDelParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		internal.AnceDelete(param.ID)
	}).FailAPIStatusForbidden(ctx)
}
