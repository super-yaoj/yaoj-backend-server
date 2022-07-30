package service

import "reflect"

type Walker = func(val reflect.Value, field reflect.StructField) error

type WalkError []error

func (r WalkError) Error() string {
	if len(r) == 0 {
		return ""
	}
	return r[0].Error()
}

func WalkStructField(v any, walker Walker) error {
	var fw = fieldWalker{errs: make([]error, 0)}
	fw.WalkValueField(reflect.ValueOf(v), reflect.StructField{}, walker)
	if len(fw.errs) == 0 {
		return nil
	}
	return WalkError(fw.errs)
}

type fieldWalker struct {
	errs []error
}

func (r *fieldWalker) WalkValueField(value reflect.Value, field reflect.StructField, walker Walker) {
	err := walker(value, field)
	if err != nil {
		r.errs = append(r.errs, err)
	}

	// 递归处理 struct

	if value.Kind() == reflect.Pointer && !value.IsNil() { // 如果是 struct 指针
		value = value.Elem()
	}

	if value.Kind() == reflect.Struct {
		tVal := value.Type()
		for i := 0; i < tVal.NumField(); i++ {
			sf := tVal.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}
			r.WalkValueField(value.Field(i), sf, walker)
		}
	}
}
