package services

import (
	"yao/internal"
)

type ClickLikeParam struct {
	Target string `body:"target"`
	UserID int    `session:"user_id" validate:"gte=0"`
	BlogID int    `body:"id" binding:"required"`
}

func ClickLike(ctx Context, param ClickLikeParam) {
	var err error
	switch param.Target {
	case "blog":
		if !internal.BLExists(param.BlogID) {
			ctx.JSONAPI(404, "", nil)
			return
		}
		err = internal.ClickLike(internal.BLOG, param.BlogID, param.UserID)
	case "comment":
		if !internal.BLCommentExists(param.BlogID) {
			ctx.JSONAPI(404, "", nil)
			return
		}
		err = internal.ClickLike(internal.COMMENT, param.BlogID, param.UserID)
	case "problem":
		if !internal.PRExists(param.BlogID) {
			ctx.JSONAPI(404, "", nil)
			return
		}
		err = internal.ClickLike(internal.PROBLEM, param.BlogID, param.UserID)
	case "contest":
		if !internal.CTExists(param.BlogID) {
			ctx.JSONAPI(404, "", nil)
			return
		}
		err = internal.ClickLike(internal.CONTEST, param.BlogID, param.UserID)
	default:
		ctx.JSONAPI(400, "invalid request", nil)
		return
	}
	if err != nil {
		ctx.ErrorAPI(err)
	}
}
