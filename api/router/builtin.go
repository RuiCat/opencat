package router

import (
	"fmt"
	"time"
)

// DefaultConfig 返回默认配置
func DefaultConfig() *RouterConfig {
	return &RouterConfig{
		MaxCallDepth:       10,
		DefaultTimeout:     30 * time.Second,
		EnableAuditLog:     true,
		EnableRecovery:     true,
		MaxFunctions:       1000,
		MaxConcurrentCalls: 100,
		EnableTriggers:     true,
		TriggerConfig: &TriggerManagerConfig{
			MaxTriggers:        100,
			EnableAsync:        true,
			MaxConcurrentFires: 50,
			EventBufferSize:    1000,
			EnableStats:        true,
		},
	}
}

// NewRouter 创建新的路由器实例
func NewRouter(config *RouterConfig) *Router {
	if config == nil {
		config = DefaultConfig()
	}
	router := &Router{
		functions:    make(map[string]*Function),
		interceptors: make(map[string]Interceptor, 0),
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
