package service

import (
	"fmt"
	"log"
	"net/http"

	"yao/config"
	"yao/service/bind"

	"github.com/gin-gonic/gin"
	"github.com/gocraft/dbr/v2"
)

type Server struct {
	*gin.Engine
	db *dbr.Connection
}

// map[method]handlerfunc
type RestApi map[string]GenHandlerFunc

// register rest api
func (r *Server) RestApi(name string, api RestApi) {
	r.OPTIONS(name, func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		ctx.Status(204)
	})

	for method, handler := range api {
		switch method {
		case "GET":
			r.GET(name, r.GinHandler(handler))
		case "POST":
			r.POST(name, r.GinHandler(handler))
		case "PATCH":
			r.PATCH(name, r.GinHandler(handler))
		case "PUT":
			r.PUT(name, r.GinHandler(handler))
		case "DELETE":
			r.DELETE(name, r.GinHandler(handler))
		default:
			panic("unknown method: " + method)
		}
	}
}

func NewServer(db *dbr.Connection) *Server {
	return &Server{
		Engine: gin.Default(),
		db:     db,
	}
}

type Context struct {
	*gin.Context
	// create a session for each business unit of execution (e.g. a web request or goworkers job)
	sess *dbr.Session
}

func (ctx *Context) DB() *dbr.Session {
	return ctx.sess
}

// APIWriteBack
func (ctx *Context) JSONAPI(statusCode int, errorMessage string, data map[string]any) {
	// log.Printf("[api] code=%d, msg=%q", statusCode, errorMessage)
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = errorMessage
	ctx.JSON(statusCode, data)
}

// RPCWriteBack
func (ctx *Context) JSONRPC(statusCode int, errorCode int, errorMessage string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = map[string]any{"code": errorCode, "message": errorMessage}
	ctx.JSON(statusCode, data)
}

// APIInternalError
func (ctx *Context) ErrorAPI(err error) {
	ctx.JSON(500, map[string]any{"_error": err.Error()})
}

// RPCInternalError
func (ctx *Context) ErrorRPC(err error) {
	ctx.JSON(500, map[string]any{"_error": map[string]any{"code": -32603, "message": err.Error()}})
}

func (ctx *Context) SetCookie(key, value string, security bool) {
	ctx.Context.SetCookie(key, value, 86400*365, "/", config.Global.FrontDomain, security, false)
}

func (ctx *Context) DeleteCookie(key string) {
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
type HandlerFunc[T any] func(ctx *Context, param T)

type GenHandlerFunc func(ctx *Context)

// 将泛型的 API handler 转化为不带泛型的 service handler
func GeneralHandler[T any](handler HandlerFunc[T]) GenHandlerFunc {
	return func(ctx *Context) {
		var data T
		err := bind.Bind(ctx.Context, &data)
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
		handler(ctx, data)
	}
}

var defaultValidator = NewServiceValidator()

// 将不带泛型的 API handler 转化为 gin handler
// route string, method string,
func (r *Server) GinHandler(handler GenHandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(&Context{
			Context: ctx,
			sess:    r.db.NewSession(nil),
		})
	}
}

// that's say, http.StatusOk shouldn't be used as a error. You should use nil instead.
type HttpStatErr int

func (r HttpStatErr) Error() string {
	return fmt.Sprint("http status code: ", int(r))
}
