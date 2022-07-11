package service

import (
	"errors"
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

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
