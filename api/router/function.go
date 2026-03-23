package router

import (
	"runtime"
	"time"
)

// Call 调用函数
// ctx: 执行上下文
// name: 函数名称
// 返回: 函数执行结果
func Call[T any](ctx *Context, name string) (zero T) {
	ctx.CallFunc(name, &zero)
	return zero
}

// CallFunc 链式调用函数
// ctx: 执行上下文
// name: 函数名称
// call: 回调函数，接收函数执行结果
func CallFunc[T any](ctx *Context, name string, call func(fn T)) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if panicValue := recover(); panicValue != nil {
			var stackTrace string
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			if n > 0 {
				stackTrace = string(buf[:n])
			} else {
				stackTrace = "无法获取堆栈信息"
			}
			ctx.LogErrorEvent(EventFunctionPanic, "函数调用发生panic", map[string]any{
				"function":  name,
				"panic":     panicValue,
				"duration":  duration.Milliseconds(),
				"stack":     stackTrace,
				"recovered": true,
			})
		}
	}()
	call(Call[T](ctx, name))
}

// CallEnhanced 增强调用函数
// ctx: 执行上下文
// name: 函数名称
// args: 函数参数
// 返回: 执行结果和错误
func CallEnhanced(ctx *Context, name string, args ...any) (any, error) {
	return ctx.Call(name, args...)
}

// CallFuncEnhanced 增强的链式调用函数
// ctx: 执行上下文
// name: 函数名称
// args: 函数参数
// handler: 结果处理回调函数
func CallFuncEnhanced(ctx *Context, name string, args []any, handler func(any, error)) {
	// 直接使用新的 Call 函数
	result, err := ctx.Call(name, args...)
	if handler != nil {
		handler(result, err)
	}
}
