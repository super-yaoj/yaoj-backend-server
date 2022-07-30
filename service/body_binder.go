package service

import (
	"fmt"
	"mime/multipart"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// 支持 "required"
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
	name, hasName := field.Tag.Lookup("body")
	bindOpt, _ := field.Tag.Lookup("binding")

	// special type: *multipart.FileHeader
	if value.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		// log.Printf("body bind file %s %q", value.Type(), field.Name)
		if !hasName {
			return false, nil
		}
		file, ferr := r.ctx.FormFile(name)
		if ferr != nil {
			return false, ferr
		}
		if file != nil {
			value.Set(reflect.ValueOf(file))
			isSet = true
		}
	} else if value.Kind() == reflect.Pointer {
		isNew := false
		vptr := value
		if value.IsNil() {
			isNew = true
			vptr = reflect.New(value.Type().Elem())
		}
		isSet, err = r.BindPostForm(vptr.Elem(), field)
		if err != nil {
			return
		}
		if isNew && isSet {
			value.Set(vptr)
		}
		return isSet, nil
	} else if value.Kind() == reflect.Struct { // 递归绑定数据，这时结构体的 tag 无意义
		tVal := value.Type()
		isSet = false
		for i := 0; i < tVal.NumField(); i++ {
			sf := tVal.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}
			isSet, err = r.BindPostForm(value.Field(i), sf)
			if err != nil {
				return
			}
		}
		return isSet, nil
	} else {
		// log.Printf("body bind %s %q", value.Type(), field.Name)

		if !hasName {
			return false, nil
		}

		if value.Kind() == reflect.Slice {
			isSet, err = false, fmt.Errorf("can't bind postform value to slice")
		} else if r.ctx.PostForm(name) == "" {
			isSet, err = false, nil
		} else {
			isSet, err = BasicBinder(r.ctx.PostForm(name)).Bind(value)
		}
	}

	if err != nil {
		return
	}
	if strings.Contains(bindOpt, "required") && !isSet {
		err = fmt.Errorf("field %q of type %q required but not binded", field.Name, value.Type())
	}
	return
}
