package service_test

import (
	"reflect"
	"testing"
	"yao/service"
)

func TestFormBinder(t *testing.T) {
	form := map[string][]string{
		"color": {"red", "blue", "green"},
		"sum":   {"5"},
		"aaa":   {"1", "2", "3", "4", "5a"},
	}
	type Data struct {
		Color []string `query:"color"`
		Sum   *int     `query:"sum"`
		Ints  []int    `query:"aaa"`
	}
	data := Data{}
	_, err := service.FormBinder(form).Bind(reflect.ValueOf(&data), reflect.StructField{})
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%+v", data)
}

type sessionGetter struct {
	data map[string]any
}

func (r *sessionGetter) Get(key any) any {
	return r.data[key.(string)]
}

func TestSessionBinder(t *testing.T) {
	type Data struct {
		Color []string `session:"color"`
		Sum   int      `session:"sum"`
		Ints  []int    `session:"aaa"`
	}
	data := Data{}
	getter := sessionGetter{data: map[string]any{
		"color": []string{"a", "b", "c"},
		"sum":   123123,
		"aaa":   []int{1, 1, 4, 5, 1, 4},
	}}
	_, err := service.SessionBinder{Session: &getter}.Bind(reflect.ValueOf(&data), reflect.StructField{})
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%+v", data)
}
