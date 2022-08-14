package services

import (
	"net/http"
	"yao/internal"
)

type BlogCreateParam struct {
	Auth
	Private int    `body:"private" validate:"gte=0,lte=1"`
	Title   string `body:"title" validate:"required,gte=1,lte=190"`
	Content string `body:"content"`
}

func BlogCreate(ctx Context, param BlogCreateParam) {
	param.NewPermit().AsNormalUser().Success(func(any) {
		id, err := internal.BLCreate(param.UserID, param.Private, param.Title, param.Content)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"id": id})
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogEditParam struct {
	Auth
	BlogID  int    `body:"blog_id" validate:"required,blogid"`
	Private int    `body:"private" validate:"gte=0,lte=1"`
	Title   string `body:"title" validate:"gte=1,lte=190"`
	Content string `body:"content"`
}

func BlogEdit(ctx Context, param BlogEditParam) {
	param.NewPermit().TryEditBlog(param.BlogID).Success(func(any) {
		err := internal.BLEdit(param.BlogID, param.Private, param.Title, param.Content)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogDelParam struct {
	Auth
	BlogID int `query:"blog_id" validate:"required,blogid"`
}

func BlogDel(ctx Context, param BlogDelParam) {
	param.NewPermit().TryEditBlog(param.BlogID).Success(func(any) {
		err := internal.BLDelete(param.BlogID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogGetParam struct {
	Auth
	BlogID int `query:"blog_id" validate:"required,blogid"`
}

func BlogGet(ctx Context, param BlogGetParam) {
	param.NewPermit().TrySeeBlog(param.BlogID).Success(func(any) {
		blog, err := internal.BLQuery(param.BlogID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"blog": blog})
		}
	}).FailAPIStatusForbidden(ctx)
}

type BlogListParam struct {
	Auth
	Page
	BlogUserID *int `query:"user_id" validate:"userid"`
}

func BlogList(ctx Context, param BlogListParam) {
	if param.BlogUserID != nil {
		blogs, err := internal.BLListUser(*param.BlogUserID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": blogs})
		}
	} else {
		if !param.CanBound() {
			ctx.JSONAPI(http.StatusBadRequest, "", nil)
			return
		}
		blogs, isfull, err := internal.BLListAll(
			param.Bound(), *param.PageSize, param.UserID, param.IsLeft(), internal.IsAdmin(param.UserGrp),
		)
		if err != nil {
			ctx.ErrorAPI(err)
			return
		}
		ctx.JSONAPI(http.StatusOK, "", map[string]any{"isfull": isfull, "data": blogs})
	}
}
