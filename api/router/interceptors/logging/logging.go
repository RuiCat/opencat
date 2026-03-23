package logging

import (
	"api/router"
	"fmt"
)

// 日志拦截器
type LoggingInterceptor struct {
	logger func(format string, args ...any)
	level  LogLevel
}

// 日志级别常量
type LogLevel int

const (
	LevelDebug LogLevel = iota // Debug level
	LevelInfo                  // Info level
	LevelWarn                  // Warning level
	LevelError                 // Error level
)

// 日志拦截器
func New(logger func(string, ...any), level LogLevel) *LoggingInterceptor {
	if logger == nil {
		logger = func(format string, args ...any) {
			fmt.Printf("[LOG] "+format+"\n", args...)
		}
	}
	return &LoggingInterceptor{
		logger: logger,
		level:  level,
	}
}

// 实现接口
func (li *LoggingInterceptor) Intercept(ctx *router.Context, call *router.InterceptorCall) error {
	if call.Result == nil && call.Err == nil {
		li.logBefore(call)
		return nil
	}
	li.logAfter(call)
	return nil
}

// 记录前
func (li *LoggingInterceptor) logBefore(call *router.InterceptorCall) {
	if li.level > LevelDebug {
		return
	}
	li.logger("函数调用开始: %s, 参数: %v, 时间: %v", call.Name, call.Args, call.Start.Format("15:04:05.000"))
}

// 记录后
func (li *LoggingInterceptor) logAfter(call *router.InterceptorCall) {
	level := LevelInfo
	if call.Err != nil {
		level = LevelError
	}
	if li.level > level {
		return
	}
	if call.Err != nil {
		li.logger("函数调用失败: %s, 耗时: %v, 错误: %v", call.Name, call.Duration, call.Err)
	} else {
		li.logger("函数调用成功: %s, 耗时: %v, 结果: %v", call.Name, call.Duration, call.Result)
	}
}

// 简单日志器
func SimpleLogger() *LoggingInterceptor {
	return New(nil, LevelInfo)
}

// 日志拦截器
func DebugLogger() *LoggingInterceptor {
	return New(nil, LevelDebug)
}

// 错误日志器
func ErrorLogger() *LoggingInterceptor {
	return New(nil, LevelError)
}
