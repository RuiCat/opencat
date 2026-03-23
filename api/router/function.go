package router

import (
	"runtime"
	"time"
)

// 函数事件常量
const (
	EventFunctionCalled                 = "function.called"                    // 函数被调用
	EventFunctionSuccess                = "function.success"                   // 函数调用成功
	EventFunctionFailed                 = "function.failed"                    // 函数调用失败
	EventFunctionPanic                  = "function.panic"                     // 函数调用panic
	EventFunctionRegistered             = "function.registered"                // 函数注册
	EventFunctionUnregistered           = "function.unregistered"              // 函数注销
	EventFunctionEnabled                = "function.enabled"                   // 函数启用
	EventFunctionDisabled               = "function.disabled"                  // 函数禁用
	EventFunctionTimeout                = "function.timeout"                   // 函数调用超时
	EventFunctionConcurrentLimit        = "function.concurrent_limit"          // 并发限制触发
	EventFunctionInterceptorStart       = "function.interceptor_start"         // 拦截器开始执行
	EventFunctionInterceptorEnd         = "function.interceptor_end"           // 拦截器执行结束
	EventFunctionInterceptorError       = "function.interceptor_error"         // 拦截器执行错误
	EventFunctionStatsUpdated           = "function.stats_updated"             // 统计信息更新
	EventFunctionContextCreated         = "function.context_created"           // 上下文创建
	EventFunctionContextDestroyed       = "function.context_destroyed"         // 上下文销毁
	EventFunctionValidationFailed       = "function.validation_failed"         // 参数验证失败
	EventFunctionRateLimited            = "function.rate_limited"              // 函数调用被限流
	EventFunctionCircuitBreakerOpen     = "function.circuit_breaker_open"      // 熔断器打开
	EventFunctionCircuitBreakerClosed   = "function.circuit_breaker_closed"    // 熔断器关闭
	EventFunctionCircuitBreakerHalfOpen = "function.circuit_breaker_half_open" // 熔断器半开
	EventFunctionRetryAttempt           = "function.retry_attempt"             // 重试尝试
	EventFunctionRetryExhausted         = "function.retry_exhausted"           // 重试耗尽
	EventFunctionCacheHit               = "function.cache_hit"                 // 缓存命中
	EventFunctionCacheMiss              = "function.cache_miss"                // 缓存未命中
	EventFunctionCacheUpdated           = "function.cache_updated"             // 缓存更新
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
