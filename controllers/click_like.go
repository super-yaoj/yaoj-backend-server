package controllers

import (
	"net/http"
	"yao/internal"
)

type ClickLikeParam struct {
	Auth
	Target string `body:"target"`
	BlogID int    `body:"id" validate:"required"`
}

func ClickLike(ctx *Context, param ClickLikeParam) {
	param.NewPermit().AsNormalUser().Success(func(a any) {
		var err error
		switch param.Target {
		case "blog":
			if !internal.BlogExists(param.BlogID) {
				ctx.JSONAPI(http.StatusNotFound, "", nil)
				return
			}
			err = internal.ClickLike(internal.BLOG, param.BlogID, param.UserID)
		case "comment":
			if !internal.BlogCommentExists(param.BlogID) {
				ctx.JSONAPI(http.StatusNotFound, "", nil)
				return
			}
			err = internal.ClickLike(internal.COMMENT, param.BlogID, param.UserID)
		case "problem":
			if !internal.ProbExists(param.BlogID) {
				ctx.JSONAPI(http.StatusNotFound, "", nil)
				return
			}
			err = internal.ClickLike(internal.PROBLEM, param.BlogID, param.UserID)
		case "contest":
			if !internal.CTExists(param.BlogID) {
				ctx.JSONAPI(http.StatusNotFound, "", nil)
				return
			}
			err = internal.ClickLike(internal.CONTEST, param.BlogID, param.UserID)
		default:
			ctx.JSONAPI(http.StatusBadRequest, "invalid request", nil)
			return
		}
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}
