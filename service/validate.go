package service

import (
	"fmt"
	"log"
	"net/mail"
	"reflect"
	"strconv"
	"strings"

	"github.com/super-yaoj/yaoj-core/pkg/utils"
)

type FieldValue struct {
	Value     reflect.Value
	Param     string // validation 参数（等于号后面的那个）
	FieldName string
}
type CheckerFunc = func(fv FieldValue) error
type validatorx struct {
	vals map[string]CheckerFunc
}

func (r *validatorx) RegisterValidation(name string, checker CheckerFunc) {
	r.vals[name] = checker
}

func (r *validatorx) Struct(o any) error {
	return WalkStructField(o, func(val reflect.Value, field reflect.StructField) error {
		alltag, ok := field.Tag.Lookup("validate")
		// log.Print(field.Name, alltag)
		if !ok {
			return nil
		}
		tags := strings.Split(alltag, ",")
		// 有关指针的 validation
		if val.Kind() == reflect.Pointer {
			if !val.IsNil() { // 若不为 nil，则对其指向的值进行校验
				val = val.Elem()
			} else { // 否则只执行 required。也就是说 required 既要求指针非 nil 也要求指向的值非零值
				if utils.FindIndex(tags, "required") != -1 {
					tags = []string{"required"}
				} else {
					tags = []string{}
				}
			}
		}
		for _, tag := range tags {
			tokens := strings.Split(tag, "=")
			if tag == "" || len(tokens) > 2 {
				return fmt.Errorf("invalid validation tag %q", tag)
			}
			chkname := tokens[0]
			chk, ok := r.vals[chkname]
			if !ok {
				return fmt.Errorf("unknown checker name %q", chkname)
			}
			var fv = FieldValue{
				Value:     val,
				FieldName: field.Name,
			}
			if len(tokens) == 2 { // no param
				fv.Param = tokens[1]
			}
			err := chk(fv)
			if err != nil {
				log.Printf("validate %q error: %q", fv.FieldName, err)
				return err
			}
		}
		return nil
	})
}

func NewValidator() validatorx {
	val := validatorx{vals: map[string]CheckerFunc{}}
	// 自带的 checker
	val.RegisterValidation("gte", func(fv FieldValue) error {
		limit, err := strconv.Atoi(fv.Param)
		if err != nil {
			panic(err)
		}
		if fv.Value.CanInt() && fv.Value.Int() >= int64(limit) {
			return nil
		}
		if fv.Value.Kind() == reflect.String && len(fv.Value.String()) >= int(limit) {
			return nil
		}
		return ValFailedErr(fv.FieldName)
	})
	val.RegisterValidation("lte", func(fv FieldValue) error {
		limit, err := strconv.Atoi(fv.Param)
		if err != nil {
			panic(err)
		}
		if fv.Value.CanInt() && fv.Value.Int() <= int64(limit) {
			return nil
		}
		if fv.Value.Kind() == reflect.String && len(fv.Value.String()) <= int(limit) {
			return nil
		}
		return ValFailedErr(fv.FieldName)
	})
	val.RegisterValidation("required", func(fv FieldValue) error {
		if fv.Value.CanInt() && fv.Value.Int() == 0 {
			return ValFailedErr("required non-zero")
		}
		switch fv.Value.Kind() {
		case reflect.String:
			if fv.Value.String() == "" {
				return ValFailedErr("required non-empty string")
			}
		case reflect.Pointer, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice:
			if fv.Value.IsNil() {
				return ValFailedErr("required non-nil")
			}
		case reflect.Bool:
			if !fv.Value.Bool() {
				return ValFailedErr("required non-false value")
			}
			fv.Value.IsNil()
		}
		return nil
	})
	val.RegisterValidation("email", func(fv FieldValue) error {
		validEmail := func(email string) bool {
			_, err := mail.ParseAddress(email)
			return err == nil
		}
		if !validEmail(fv.Value.String()) {
			return ValFailedErr("invalid email")
		}
		return nil
	})
	return val
}
