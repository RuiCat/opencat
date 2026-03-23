package router

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
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
	EventTriggerRegistered              = "trigger.registered"                 // 触发器注册
	EventTriggerUnregistered            = "trigger.unregistered"               // 触发器注销
	EventTriggerEnabled                 = "trigger.enabled"                    // 触发器启用
	EventTriggerDisabled                = "trigger.disabled"                   // 触发器禁用
	EventTriggerFired                   = "trigger.fired"                      // 触发器触发
	EventTriggerError                   = "trigger.error"                      // 触发器错误
)

// DefaultConfig 返回默认的路由器配置
// 返回: 默认配置实例
func DefaultConfig() *RouterConfig {
	return &RouterConfig{
		MaxCallDepth:       10,
		DefaultTimeout:     30 * time.Second,
		EnableAuditLog:     true,
		EnableRecovery:     true,
		MaxFunctions:       1000,
		MaxConcurrentCalls: 100,
		EnableTriggers:     true,
		EnableAsyncEvents:  true,
		TriggerConfig: &TriggerManagerConfig{
			MaxTriggers:        100,
			EnableAsync:        false,
			MaxConcurrentFires: 50,
			EventBufferSize:    1000,
			EnableStats:        true,
		},
	}
}

// NewBasicRouter 创建基础路由器
// config: 路由器配置，如果为nil则使用默认配置
// 返回: 路由器实例
func NewBasicRouter(config *RouterConfig) *Router {
	if config == nil {
		config = DefaultConfig()
	}
	router := &Router{
		functions: make(map[string]*Function),
		eventBus: &EventBus{
			subscribers: make(map[string][]EventHandler),
		},
		config:        config,
		callSemaphore: make(chan struct{}, config.MaxConcurrentCalls),
		stats: &RouterStats{
			StartTime: time.Now(),
		},
	}
	if config.EnableTriggers {
		if config.TriggerConfig == nil {
			config.TriggerConfig = &TriggerManagerConfig{
				MaxTriggers:        100,
				EnableAsync:        true,
				MaxConcurrentFires: 50,
				EventBufferSize:    1000,
				EnableStats:        true,
			}
		}
		router.triggerManager = &TriggerManager{
			triggers: make(map[string]*Trigger),
			eventBus: router.eventBus,
			stats: &TriggerManagerStats{
				StartTime: time.Now(),
			},
			config: config.TriggerConfig,
		}
	}
	return router
}

// NewRouter 创建路由器
func NewRouter(config *RouterConfig) *Router {
	if config == nil {
		config = DefaultConfig()
	}
	router := NewBasicRouter(config)
	router.eventPublisher = NewRouterEventPublisher(router, config.EnableAsyncEvents)
	return router
}

// NewContext 创建新的上下文
// sessionID: 会话ID
// agentID: 调用者ID
// router: 路由器接口
// 返回: 新的上下文实例
func NewContext(sessionID, agentID string, router *Router) *Context {
	var timeout time.Duration
	if config := router.GetConfig(); config != nil {
		timeout = config.DefaultTimeout
	} else {
		timeout = 30 * time.Second
	}
	context := &Context{
		SessionID:   sessionID,
		AgentID:     agentID,
		TraceID:     fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		StartTime:   time.Now(),
		Router:      router,
		Timeout:     timeout,
		CallDepth:   0,
		wrappers:    sync.Map{},
		interceptor: NewInterceptorChain(),
		recoveryHandler: func(ctx *Context, block *DataBlock, panicValue any) *Result {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:      "PANIC_RECOVERED",
					Message:   fmt.Sprintf("函数执行panic: %v", panicValue),
					Recovered: true,
					Retryable: false,
				},
				TraceID: block.TraceID,
			}
		},
	}
	context.Values = map[string]any{
		"router":  router,
		"context": context,
	}
	// 发布上下文创建事件
	router.PublishEventName(EventFunctionContextCreated, map[string]any{
		"session_id": sessionID,
		"agent_id":   agentID,
		"trace_id":   context.TraceID,
		"timestamp":  time.Now().UnixNano(),
	})
	return context
}

// NewSafeAsyncExecutor 创建安全的异步执行器
func NewSafeAsyncExecutor(maxWorkers int) *SafeAsyncExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 100
	}
	return &SafeAsyncExecutor{
		workerPool: NewWorkerPool(maxWorkers),
		stop:       make(chan struct{}),
	}
}

// NewEventPublisher 创建事件发布器
func NewEventPublisher(eventBus *EventBus, enableAsync bool) *EventPublisher {
	var executor AsyncExecutor
	if enableAsync {
		executor = NewSafeAsyncExecutor(50)
	} else {
		executor = &SyncExecutor{}
	}
	return &EventPublisher{
		executor:    executor,
		eventBus:    eventBus,
		enableAsync: enableAsync,
	}
}

// NewRouterEventPublisher 创建路由器事件发布器
func NewRouterEventPublisher(router *Router, enableAsync bool) *RouterEventPublisher {
	return &RouterEventPublisher{
		router:          router,
		eventPublisher:  NewEventPublisher(router.eventBus, enableAsync),
		triggerExecutor: &SyncExecutor{},
	}
}

// NewInterceptorChain 创建新的拦截器链
func NewInterceptorChain() *InterceptorChain {
	return &InterceptorChain{}
}

// NewInterceptorCall 创建新的拦截器调用
func NewInterceptorCall(name string, args []any) *InterceptorCall {
	return &InterceptorCall{
		Name:     name,
		Args:     args,
		Start:    time.Now(),
		Metadata: make(map[string]any),
	}
}

// NewEventBatcher 创建事件批处理器
func NewEventBatcher(batchSize int, timeout time.Duration, handler func([]*Event)) *EventBatcher {
	if batchSize <= 0 {
		batchSize = 10
	}
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}
	batcher := &EventBatcher{
		events:    make(chan *Event, batchSize*10),
		batchSize: batchSize,
		timeout:   timeout,
		handler:   handler,
		stop:      make(chan struct{}),
	}
	batcher.wg.Add(1)
	go batcher.process()
	return batcher
}

// NewWorkerPool 创建工作池
// maxWorkers: 最大工作协程数
// 返回: WorkerPool实例
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 100
	}
	return &WorkerPool{
		workers: make(chan struct{}, maxWorkers),
		stop:    make(chan struct{}),
	}
}

// SafeGo 安全的协程执行
// fn: 要执行的函数
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SAFE-GO] goroutine panic recovered: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}
