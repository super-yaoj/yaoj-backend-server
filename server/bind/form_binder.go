package bind

import (
	"fmt"
	"reflect"
	"strings"
)

// https://stackoverflow.com/questions/724526/how-to-pass-multiple-parameters-in-a-querystring
type FormBinder map[string][]string

// 注意不支持 map, array！
// 支持 required
func (r FormBinder) Bind(value reflect.Value, field reflect.StructField) (isSet bool, err error) {
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
			isSet, err = r.Bind(value.Field(i), sf)
			if err != nil {
				return false, err
			}
		}
		return isSet, nil
	}

	name, ok := field.Tag.Lookup("query")
	if !ok {
		return false, nil
	}
	bindOpt, _ := field.Tag.Lookup("binding")
	// log.Printf("form bind %s %q %q", value.Type(), field.Name, bindOpt)

	if value.Kind() == reflect.Slice {
		if !value.CanSet() {
			panic("value can't set")
		}

		isSet, err = arrayBinder(r[name]).Bind(value)
	} else if r[name] == nil || len(r[name]) == 0 {
		isSet, err = false, nil
	} else {
		isSet, err = BasicBinder(r[name][0]).Bind(value)
	}

	if err != nil {
		return
	}
	if strings.Contains(bindOpt, "required") && !isSet {
		err = fmt.Errorf("field %q of type %q required but not binded", field.Name, value.Type())
	}
	return
}
