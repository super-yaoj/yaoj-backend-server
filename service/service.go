package service

import (
	"fmt"
	"log"
	"net/http"

	"yao/config"
	"yao/internal"
	"yao/service/bind"

	"github.com/gin-gonic/gin"
)

type Context struct {
	*gin.Context
}

// APIWriteBack
func (ctx Context) JSONAPI(statusCode int, errorMessage string, data map[string]any) {
	// log.Printf("[api] code=%d, msg=%q", statusCode, errorMessage)
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = errorMessage
	ctx.JSON(statusCode, data)
}

// RPCWriteBack
func (ctx Context) JSONRPC(statusCode int, errorCode int, errorMessage string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = map[string]any{"code": errorCode, "message": errorMessage}
	ctx.JSON(statusCode, data)
}

// APIInternalError
func (ctx Context) ErrorAPI(err error) {
	ctx.JSON(500, map[string]any{"_error": err.Error()})
}

// RPCInternalError
func (ctx Context) ErrorRPC(err error) {
	ctx.JSON(500, map[string]any{"_error": map[string]any{"code": -32603, "message": err.Error()}})
}

func (ctx Context) SetCookie(key, value string, security bool) {
	ctx.Context.SetCookie(key, value, 86400*365, "/", config.Global.FrontDomain, security, false)
}

func (ctx Context) DeleteCookie(key string) {
	ctx.Context.SetCookie(key, "", -1, "/", config.Global.FrontDomain, false, false)
}

// Universal Web API Parameter Design
//
// 参数可能的来源：query, body, session, （后面三个尚未支持）uri, header, cookie
//
// 对于 body，会自动根据 ContentType 来解析字段，目前已支持：
//
//	application/x-www-form-urlencoded
//	multipart/form-data
//
// session: github.com/gin-contrib/sessions
//
// 注意绑定 session 数据时字段（key）的类型必须为 string，且 session 中存的类型
// 须与被绑定数据的类型完全一致
//
// 请注意用于绑定的数据尽量不要出现指针值（主要指 session）
// T 为参数类型结构体
//
// 注意 binding:"required" 并非适用于所有来源的绑定，目前只适用于 query 和 body
//
// validate
//
// required:
// 对于引用类型，要求不能是 nil。
// 对于其他值，要求不能是零值。
// 对于指针要求既不是 nil 也不是零值。
type StdHandlerFunc[T any] func(ctx Context, param T)

var defaultValidator = NewValidator()

// 将标准化的 API handler 转化为 gin handler
// route string, method string,
func GinHandler[T any](handler StdHandlerFunc[T]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var data T
		err := bind.Bind(ctx, &data)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("[bind]: %s", err)
			return
		}
		// pp.Println(data)
		err = defaultValidator.Struct(data)
		if err != nil {
			errs := err.(WalkError)
			switch terr := errs[0].(type) {
			case HttpStatErr:
				ctx.JSON(int(terr), gin.H{"error": "validation failed"})
			default:
				ctx.JSON(http.StatusBadRequest, gin.H{"error": terr.Error()})
			}
			log.Printf("[validate]: %s", err)
			return
		}
		handler(Context{Context: ctx}, data)
	}
}

// that's say, http.StatusOk shouldn't be used as a error. You should use nil instead.
type HttpStatErr int

func (r HttpStatErr) Error() string {
	return fmt.Sprint("http status code: ", int(r))
}

type ValFailedErr string

func (r ValFailedErr) Error() string {
	return fmt.Sprint("validation failed: ", string(r))
}

func init() {
	// check whether the user_group belongs to admin
	defaultValidator.RegisterValidation("admin", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.IsAdmin(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusForbidden)
		}
		return nil // ok
	})
	// check whether the problem id exist in database
	defaultValidator.RegisterValidation("probid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.ProbExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("submid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.SubmExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("ctstid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.CTExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("prmsid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.PermExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("userid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.UserExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("blogid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.BlogExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	defaultValidator.RegisterValidation("cmntid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.BlogCommentExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	// authentication
	defaultValidator.RegisterValidation("pagecanbound", func(fv FieldValue) error {
		page, ok := fv.Value.Interface().(Page)
		if !ok {
			return ValFailedErr("invalid Page struct")
		}
		// pp.Print(page, page.CanBound())
		if !page.CanBound() {
			return HttpStatErr(http.StatusBadRequest)
		}
		return nil
	})
}
