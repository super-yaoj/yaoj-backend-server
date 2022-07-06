package services

import (
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func BLCanSee(ctx *gin.Context, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author, private from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return ISAdmin(ctx) || GetUserId(ctx) == blog.Author || !blog.Private
	}
}

func BLCanEdit(ctx *gin.Context, blog_id int) bool {
	var blog internal.Blog
	err := libs.DBSelectSingle(&blog, "select blog_id, author from blogs where blog_id=?", blog_id)
	if err != nil {
		return false
	} else {
		return ISAdmin(ctx) || GetUserId(ctx) == blog.Author
	}
}

func BLCreate(ctx *gin.Context) {
	user_id := GetUserId(ctx)
	private, ok := libs.PostIntRange(ctx, "private", 0, 1)
	if !ok {
		return
	}
	title := ctx.PostForm("title")
	if !internal.BLValidTitle(title) {
		libs.APIWriteBack(ctx, 400, "invalid title", nil)
		return
	}
	content := ctx.PostForm("content")
	id, err := internal.BLCreate(user_id, private, title, content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
	}
}

func BLEdit(ctx *gin.Context) {
	id, ok := libs.PostInt(ctx, "blog_id")
	if !ok {
		return
	}
	if !internal.BLExists(id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	private, ok := libs.PostIntRange(ctx, "private", 0, 1)
	if !ok {
		return
	}
	if !BLCanEdit(ctx, id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	title := ctx.PostForm("title")
	if !internal.BLValidTitle(title) {
		libs.APIWriteBack(ctx, 400, "invalid title", nil)
		return
	}
	content := ctx.PostForm("content")
	err := internal.BLEdit(id, private, title, content)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func BLDelete(ctx *gin.Context) {
	id, ok := libs.GetInt(ctx, "blog_id")
	if !ok {
		return
	}
	if !BLCanEdit(ctx, id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	err := internal.BLDelete(id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func BLQuery(ctx *gin.Context) {
	id, ok := libs.GetInt(ctx, "blog_id")
	if !ok {
		return
	}
	if !internal.BLExists(id) {
		libs.APIWriteBack(ctx, 404, "", nil)
		return
	}
	if !BLCanSee(ctx, id) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	blog, err := internal.BLQuery(id, GetUserId(ctx))
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		ret, _ := libs.Struct2Map(blog)
		libs.APIWriteBack(ctx, 200, "", ret)
	}
}

func BLList(ctx *gin.Context) {
	_, is_user := ctx.GetQuery("user_id")
	if is_user {
		user_id, ok := libs.GetInt(ctx, "user_id")
		if !ok {
			return
		}
		blogs, err := internal.BLListUser(user_id, GetUserId(ctx))
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": blogs})
		}
	} else {
		pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
		if !ok {
			return
		}
		_, isleft := ctx.GetQuery("left")
		bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
		if !ok {
			return
		}
		blogs, isfull, err := internal.BLListAll(bound, pagesize, GetUserId(ctx), isleft, ISAdmin(ctx))
		if err != nil {
			libs.APIInternalError(ctx, err)
			return
		}
		libs.APIWriteBack(ctx, 200, "", map[string]any{"isfull": isfull, "data": blogs})
	}
}
