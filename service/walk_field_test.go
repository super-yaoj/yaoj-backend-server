package service_test

import (
	"reflect"
	"testing"
	"yao/service"
)

func TestWalkStructField(t *testing.T) {
	type Obj struct {
		A, B int
		C    struct {
			D string
		}
	}

	err := service.WalkStructField(Obj{}, func(val reflect.Value, field reflect.StructField) error {
		t.Logf("field: %s", field.Name)
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}
