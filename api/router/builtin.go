package router

import (
	"fmt"
	"time"
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
