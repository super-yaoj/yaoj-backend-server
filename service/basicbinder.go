package service

import (
	"fmt"
	"reflect"
	"strconv"
)

// Binder 自身带有数据，可以把自己的数据绑定到给定的 value 上
type FieldBinder interface {
	Bind(value reflect.Value, field reflect.StructField) (isSet bool, err error)
}

// basic types
type BasicBinder string

func (r BasicBinder) Bind(value reflect.Value) (bool, error) {
	switch value.Kind() {
	case reflect.Int:
		val, _ := strconv.ParseInt(string(r), 10, 32)
		value.SetInt(val)
		return true, nil
	case reflect.Int8:
		val, _ := strconv.ParseInt(string(r), 10, 8)
		value.SetInt(val)
		return true, nil
	case reflect.Int16:
		val, _ := strconv.ParseInt(string(r), 10, 16)
		value.SetInt(val)
		return true, nil
	case reflect.Int32:
		val, _ := strconv.ParseInt(string(r), 10, 32)
		value.SetInt(val)
		return true, nil
	case reflect.Int64:
		val, _ := strconv.ParseInt(string(r), 10, 64)
		value.SetInt(val)
		return true, nil
	case reflect.Uint:
		val, _ := strconv.ParseUint(string(r), 10, 32)
		value.SetUint(val)
		return true, nil
	case reflect.Uint8:
		val, _ := strconv.ParseUint(string(r), 10, 8)
		value.SetUint(val)
		return true, nil
	case reflect.Uint16:
		val, _ := strconv.ParseUint(string(r), 10, 16)
		value.SetUint(val)
		return true, nil
	case reflect.Uint32:
		val, _ := strconv.ParseUint(string(r), 10, 32)
		value.SetUint(val)
		return true, nil
	case reflect.Uint64:
		val, _ := strconv.ParseUint(string(r), 10, 64)
		value.SetUint(val)
		return true, nil
	case reflect.String:
		value.SetString(string(r))
		return true, nil
	case reflect.Bool:
		val, _ := strconv.ParseBool(string(r))
		value.SetBool(val)
		return true, nil
	case reflect.Float32:
		val, _ := strconv.ParseFloat(string(r), 32)
		value.SetFloat(val)
		return true, nil
	case reflect.Float64:
		val, _ := strconv.ParseFloat(string(r), 64)
		value.SetFloat(val)
		return true, nil
	}
	return false, fmt.Errorf("invalid int type %q", value.Kind().String())
}

type arrayBinder []string

func (r arrayBinder) Bind(value reflect.Value) (isSet bool, err error) {
	if value.Kind() != reflect.Slice {
		return false, fmt.Errorf("invalid array type %q", value.Kind().String())
	}
	slice := reflect.MakeSlice(value.Type(), len(r), len(r))
	for i, str := range r {
		isSet, err = BasicBinder(str).Bind(slice.Index(i))
		if err != nil {
			return
		}
	}
	value.Set(slice)
	return
}
