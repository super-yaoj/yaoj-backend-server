package service

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

type BodyBinder struct {
	ctx *gin.Context
}

func (r BodyBinder) Bind(value reflect.Value, field reflect.StructField) (isSet bool, err error) {
	ctype := r.ctx.ContentType()
	if ctype == "" { // no body
		return false, nil
	}
	if strings.Contains(ctype, "multipart/form-data") {
		return r.BindPostForm(value, field)
	}
	if strings.Contains(ctype, "application/x-www-form-urlencoded") {
		return r.BindPostForm(value, field)
	}
	return false, fmt.Errorf("unknown content type %q", ctype)
}

func (r BodyBinder) BindPostForm(value reflect.Value, field reflect.StructField) (isSet bool, err error) {
	log.Printf("body bind %s %q", value.Type(), field.Name)
	if value.Kind() == reflect.Pointer {
		isNew := false
		vptr := value
		if value.IsNil() {
			isNew = true
			vptr = reflect.New(value.Type().Elem())
		}
		isSet, err = r.BindPostForm(vptr.Elem(), field)
		if err != nil {
			return false, err
		}
		if isNew && isSet {
			value.Set(vptr)
		}
		return isSet, nil
	}

	// 递归绑定数据，这时结构体的 tag 无意义
	if value.Kind() == reflect.Struct {
		tVal := value.Type()
		isSet = false
		for i := 0; i < tVal.NumField(); i++ {
			sf := tVal.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}
			name, ok := sf.Tag.Lookup("body")
			if !ok || r.ctx.PostForm(name) == "" {
				continue
			}

			isSet, err = r.BindPostForm(value.Field(i), sf)
			if err != nil {
				return false, err
			}

		}
		return isSet, nil
	}

	name, ok := field.Tag.Lookup("body")
	if !ok || r.ctx.PostForm(name) == "" {
		return false, nil
	}

	if value.Kind() == reflect.Slice {
		return false, fmt.Errorf("can't bind postform value to slice")
	}
	isSet, err = BasicBinder(r.ctx.PostForm(name)).Bind(value)
	if err != nil {
		return false, err
	}
	return isSet, nil
}
