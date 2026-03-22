package router

import (
	"reflect"
	"runtime"
	"time"
)

// 函数调用事件常量
const (
	EventFunctionCalled  = "function.called"  // 函数被调用
	EventFunctionSuccess = "function.success" // 函数调用成功
	EventFunctionFailed  = "function.failed"  // 函数调用失败
	EventFunctionPanic   = "function.panic"   // 函数调用panic
)

// Call 调用
func Call[T any](ctx *Context, name string) (zero T) {
	startTime := time.Now()
	fn := ctx.Router.GetFunction(name)
	// 发布函数调用开始事件
	ctx.Router.PublishEventName(EventFunctionCalled, map[string]any{
		"function":  name,
		"caller":    ctx.AgentID,
		"trace_id":  ctx.TraceID,
		"depth":     ctx.GetCallDepth(),
		"timestamp": startTime.UnixNano(),
	})
	switch {
	case fn == nil:
		ctx.LogError("函数调用失败", map[string]any{"name": name})
		// 发布函数调用失败事件
		ctx.Router.PublishEventName(EventFunctionFailed, map[string]any{
			"function": name,
			"error":    "函数不存在",
			"trace_id": ctx.TraceID,
			"duration": time.Since(startTime).Milliseconds(),
			"reason":   "function_not_found",
		})
		return zero
	case fn.IsMethod:
		v := ctx.GetValue(fn.Namespace)
		if v == nil {
			ctx.LogError("函数命名空间不存在", map[string]any{"name": name, "namespace": fn.Namespace})
			// 发布函数调用失败事件
			ctx.Router.PublishEventName(EventFunctionFailed, map[string]any{
				"function":  name,
				"namespace": fn.Namespace,
				"error":     "命名空间不存在",
				"trace_id":  ctx.TraceID,
				"duration":  time.Since(startTime).Milliseconds(),
				"reason":    "namespace_not_found",
			})
			return zero
		}
		targetType := reflect.TypeOf(zero)
		if targetType == nil || targetType.Kind() != reflect.Func {
			ctx.LogError("目标类型必须是函数类型", map[string]any{"name": name})
			// 发布函数调用失败事件
			ctx.Router.PublishEventName(EventFunctionFailed, map[string]any{
				"function": name,
				"error":    "目标类型不是函数",
				"trace_id": ctx.TraceID,
				"duration": time.Since(startTime).Milliseconds(),
				"reason":   "invalid_target_type",
			})
			return zero
		}
		originalFn := reflect.ValueOf(fn.Function)
		if originalFn.Kind() != reflect.Func {
			ctx.LogError("注册的函数无效", map[string]any{"name": name})
			// 发布函数调用失败事件
			ctx.Router.PublishEventName(EventFunctionFailed, map[string]any{
				"function": name,
				"error":    "注册的函数无效",
				"trace_id": ctx.TraceID,
				"duration": time.Since(startTime).Milliseconds(),
				"reason":   "invalid_function",
			})
			return zero
		}
		wrappedFn := reflect.MakeFunc(targetType, func(args []reflect.Value) []reflect.Value {
			// 发布函数调用成功事件
			ctx.Router.PublishEventName(EventFunctionSuccess, map[string]any{
				"function":  name,
				"namespace": fn.Namespace,
				"is_method": true,
				"trace_id":  ctx.TraceID,
				"duration":  time.Since(startTime).Milliseconds(),
				"caller":    ctx.AgentID,
			})
			return originalFn.Call(append([]reflect.Value{reflect.ValueOf(v)}, args...))
		})
		return wrappedFn.Interface().(T)
	default:
		// 发布函数调用成功事件
		ctx.Router.PublishEventName(EventFunctionSuccess, map[string]any{
			"function":  name,
			"is_method": false,
			"trace_id":  ctx.TraceID,
			"duration":  time.Since(startTime).Milliseconds(),
			"caller":    ctx.AgentID,
		})
		return (fn.Function).(T)
	}
}

// CallFunc 链式调用
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
			// 发布函数调用 panic 事件
			ctx.Router.PublishEventName(EventFunctionPanic, map[string]any{
				"function":  name,
				"panic":     panicValue,
				"trace_id":  ctx.TraceID,
				"duration":  duration.Milliseconds(),
				"caller":    ctx.AgentID,
				"stack":     stackTrace,
				"recovered": true,
				"timestamp": time.Now().UnixNano(),
			})
			// 记录错误日志
			ctx.LogError("函数调用发生panic", map[string]any{
				"function": name,
				"panic":    panicValue,
				"trace_id": ctx.TraceID,
				"duration": duration.Milliseconds(),
				"stack":    stackTrace,
			})
		}
	}()
	call(Call[T](ctx, name))
}
