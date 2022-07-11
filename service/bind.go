package service

import (
	"errors"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/k0kubun/pp"
)

// Universal Web API Parameter Design
// 参数可能的来源：query, body, session, （后面三个尚未支持）uri, header, cookie
// 对于 body，会自动根据 ContentType 来解析字段，目前已支持：
//
//   application/x-www-form-urlencoded
//   multipart/form-data
//
// session: github.com/gin-contrib/sessions
// 注意绑定 session 数据时字段（key）的类型必须为 string，且 session 中存的类型
// 须与被绑定数据的类型完全一致
//
// T 为参数类型结构体
type StdHandlerFunc[T any] func(ctx *gin.Context, param T)

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
		handler(ctx, data)
	}
}

// "query"
func bindQuery(ctx *gin.Context, ptr any) (isSet bool, err error) {
	form := ctx.Request.URL.Query()
	ptrVal := reflect.ValueOf(ptr)
	// log.Printf("bind query")
	// pp.Print(form)

	isSet, err = FormBinder(form).Bind(ptrVal, reflect.StructField{})
	return
}

// "session"
func bindSession(ctx *gin.Context, ptr any) (isSet bool, err error) {
	sess := sessions.Default(ctx)
	ptrVal := reflect.ValueOf(ptr)

	isSet, err = SessionBinder{Session: sess}.Bind(ptrVal, reflect.StructField{})
	return
}

// "body"
func bindBody(ctx *gin.Context, ptr any) (isSet bool, err error) {
	ptrVal := reflect.ValueOf(ptr)
	isSet, err = BodyBinder{ctx: ctx}.Bind(ptrVal, reflect.StructField{})
	return
}

// obj 必须是一个结构体变量的指针
func Bind(ctx *gin.Context, ptr any) error {
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.Kind() != reflect.Ptr {
		return errors.New("invalid ptr type")
	}
	if _, err := bindQuery(ctx, ptr); err != nil {
		return err
	}
	if _, err := bindSession(ctx, ptr); err != nil {
		return err
	}
	if _, err := bindBody(ctx, ptr); err != nil {
		return err
	}
	return nil
}
