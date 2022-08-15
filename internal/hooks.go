package internal

import (
	"fmt"
	"reflect"
	"sync"
)

type hookValue struct {
	in     []reflect.Type
	callee []any
}

var (
	hooks = make(map[string]*hookValue)
	hookLock = sync.Mutex{}
)

//将此函数的调用者制作为一个钩子，mode 为钩子事件（AfterSubmCreate 等），in 为钩子传递的参数。
//
//例如 func MyFunc() { hook.Register("onMyFunc", true) }，
//你可以在其他位置使用 hook.Listen("onMyFunc", func(a bool) { fmt.Println(a) }) 来使用这个钩子，并且此后每次执行 MyFunc 时都会打印出 a=true。
//
//此函数使用 reflect 确保 register 给定的输入与 listen 的参数一致
func Register(mode string, in ...any) {
	hookLock.Lock()
	defer hookLock.Unlock()
	value, ok := hooks[mode]
	if !ok {
		value = &hookValue{in: nil, callee: []any{}}
		hooks[mode] = value
	}
	if value.in == nil {
		value.in = make([]reflect.Type, len(in))
		for key := range in {
			value.in[key] = reflect.TypeOf(in[key])
		}
	} else {
		check := true
		if len(in) != len(value.in) {
			check = false
		} else {
			for key := range in {
				if in[key] != value.in[key] {
					check = false
					break
				}
			}
		}
		if (!check) {
			fmt.Println("error on hook registration: in type error")
			return
		}
	}
	//run callees
	args := make([]reflect.Value, len(in))
	for key := range in {
		args[key] = reflect.ValueOf(in[key])
	}
	for _, f := range value.callee {
		reflect.ValueOf(f).Call(args)
	}
}

//增添一个监听事件，mode 为钩子事件（AfterSubmCreate 等），callee 为钩子执行的函数，需要与钩子给出的参数对应。
func Listen(mode string, callee any) {
	hookLock.Lock()
	defer hookLock.Unlock()
	value, ok := hooks[mode]
	if !ok {
		value = &hookValue{in: nil, callee: []any{}}
		hooks[mode] = value
	}
	value.callee = append(value.callee, callee)
}