package service

import (
	"log"
	"net/http"
	"yao/libs"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/k0kubun/pp"
)

type Context struct {
	*gin.Context
}

// APIWriteBack
func (ctx Context) JSONAPI(statusCode int, errorMessage string, data map[string]any) {
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

// Universal Web API Parameter Design
//
// 参数可能的来源：query, body, session, （后面三个尚未支持）uri, header, cookie
//
// 对于 body，会自动根据 ContentType 来解析字段，目前已支持：
//
//   application/x-www-form-urlencoded
//   multipart/form-data
//
// session: github.com/gin-contrib/sessions
//
// 注意绑定 session 数据时字段（key）的类型必须为 string，且 session 中存的类型
// 须与被绑定数据的类型完全一致
//
// 请注意用于绑定的数据尽量不要出现指针值（主要指 session）
// T 为参数类型结构体
type StdHandlerFunc[T any] func(ctx Context, param T)

var defaultValidator = validator.New()

// 将标准化的 API handler 转化为 gin handler
// route string, method string,
func GinHandler[T any](handler StdHandlerFunc[T]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var data T
		err := Bind(ctx, &data)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("[bind]: %s", err)
			return
		}
		pp.Println(data)
		err = defaultValidator.Struct(data)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("[validate]: %s", err)
			return
		}
		handler(Context{Context: ctx}, data)
	}
}

func init() {
	// check whether the user_group belongs to admin
	defaultValidator.RegisterValidation("admin", func(fl validator.FieldLevel) bool {
		if !fl.Field().CanInt() {
			return false
		}
		if !libs.IsAdmin(int(fl.Field().Int())) {
			return false
		}
		return true
	})
}
