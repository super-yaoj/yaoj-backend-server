package server_test

import (
	"reflect"
	"testing"
	"yao/server"
)

func TestWalkStructField(t *testing.T) {
	type Obj struct {
		A, B int
		C    struct {
			D string
		}
	}

	err := server.WalkStructField(Obj{}, func(val reflect.Value, field reflect.StructField) error {
		t.Logf("field: %s", field.Name)
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}
