package service

import (
	"fmt"
	"reflect"
)

type SessionGetter interface {
	Get(key interface{}) interface{}
}
type SessionBinder struct {
	Session SessionGetter
}

func (r SessionBinder) Bind(value reflect.Value, field reflect.StructField) (isSet bool, err error) {
	if value.Kind() == reflect.Pointer {
		isNew := false
		vptr := value
		if value.IsNil() {
			isNew = true
			vptr = reflect.New(value.Type().Elem())
		}
		isSet, err = r.Bind(vptr.Elem(), field)
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
			name, ok := sf.Tag.Lookup("session")
			if !ok || r.Session.Get(name) == nil {
				continue
			}

			isSet, err = r.Bind(value.Field(i), sf)
			if err != nil {
				return false, err
			}

		}
		return isSet, nil
	}

	name, ok := field.Tag.Lookup("session")
	if !ok || r.Session.Get(name) == nil {
		return false, nil
	}

	sessVal := reflect.ValueOf(r.Session.Get(name))

	if value.Type() != sessVal.Type() {
		return false, fmt.Errorf("error assign type %q with %q", value.Type().String(), sessVal.Type().String())
	}
	// ptrSessVal := reflect.New(sessVal.Type())
	// ptrSessVal.Elem().Set(sessVal)

	value.Set(sessVal)
	if err != nil {
		return false, err
	}
	return isSet, nil
}
