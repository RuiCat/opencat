package router

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Router 路由器
type Router struct {
	mu              sync.RWMutex
	functions       map[string]*Function
	eventBus        *EventBus
	config          *RouterConfig
	callSemaphore   chan struct{}
	stats           *RouterStats
	triggerManager  *TriggerManager
	eventPublisher  *RouterEventPublisher
	recoveryHandler RecoveryHandler
}

// NewRouter 创建路由器
func NewRouter(config *RouterConfig) *Router {
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
	router.eventPublisher = NewRouterEventPublisher(router, config.EnableAsyncEvents)
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

// Register 注册用户自定义函数
func (r *Router) Register(name, description, namespace string, inputSchema, outputSchema []string, fun any) error {
	return r.RegisterFunction(&Function{
		Name:         name,
		Description:  description,
		Namespace:    namespace,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Function:     fun,
		Builtin:      false,
		Enabled:      true,
		CreatedAt:    time.Now(),
		Stats:        &FunctionStats{},
	})
}

// RegisterFunction 注册函数
func (r *Router) RegisterFunction(fn *Function) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.functions) >= r.config.MaxFunctions {
		return fmt.Errorf("达到最大函数数量限制: %d", r.config.MaxFunctions)
	}
	if fn.Name == "" {
		return fmt.Errorf("函数名称不能为空")
	}
	if _, exists := r.functions[fn.Name]; exists {
		return fmt.Errorf("函数已存在: %s", fn.Name)
	}
	if fn.Function == nil {
		return fmt.Errorf("函数实现不能为空")
	}
	val := reflect.ValueOf(fn.Function)
	if val.Kind() != reflect.Func {
		return fmt.Errorf("Function 字段必须是函数类型，当前类型: %v", val.Kind())
	}
	if fn.CreatedAt.IsZero() {
		fn.CreatedAt = time.Now()
	}
	if fn.Stats == nil {
		fn.Stats = &FunctionStats{}
	}
	tys := val.Type()
	if numIn := tys.NumIn(); numIn != len(fn.InputSchema) {
		firstArg := tys.In(0)
		if fn.Namespace != "" && numIn == len(fn.InputSchema)+1 &&
			(firstArg.Kind() == reflect.Struct || (firstArg.Kind() == reflect.Ptr && firstArg.Elem().Kind() == reflect.Struct)) {
			fn.IsMethod = true
		} else {
			return fmt.Errorf("函数参数数量不匹配: 函数有 %d 个参数，InputSchema 有 %d 个参数",
				numIn, len(fn.InputSchema))
		}
	}
	r.functions[fn.Name] = fn
	r.stats.FunctionCount = len(r.functions)
	builder := NewEventData().WithFunctionInfo(fn)
	r.eventPublisher.PublishEventName(EventFunctionRegistered, builder.Build())
	return nil
}

// Unregister 注销用户自定义函数
func (r *Router) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if name == "" {
		return fmt.Errorf("函数名称不能为空")
	}
	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}
	if fn.Builtin {
		return fmt.Errorf("不能注销内置函数: %s", name)
	}
	delete(r.functions, name)
	r.stats.FunctionCount = len(r.functions)
	builder := NewEventData().
		With("name", name).
		With("namespace", fn.Namespace).
		With("description", fn.Description).
		With("builtin", fn.Builtin)
	r.eventPublisher.PublishEventName(EventFunctionUnregistered, builder.Build())
	return nil
}

// GetFunction 获取函数
func (r *Router) GetFunction(name string) *Function {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.functions[name]
}

// ListFunctions 列出所有函数
func (r *Router) ListFunctions() map[string]*Function {
	r.mu.RLock()
	defer r.mu.RUnlock()
	functions := make(map[string]*Function, len(r.functions))
	for k, v := range r.functions {
		functions[k] = v
	}
	return functions
}

// EnableFunction 启用函数
func (r *Router) EnableFunction(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}
	fn.Enabled = true
	builder := NewEventData().WithFunctionInfo(fn)
	r.eventPublisher.PublishEventName(EventFunctionEnabled, builder.Build())
	return nil
}

// DisableFunction 禁用函数
func (r *Router) DisableFunction(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}
	fn.Enabled = false
	builder := NewEventData().WithFunctionInfo(fn)
	r.eventPublisher.PublishEventName(EventFunctionDisabled, builder.Build())
	return nil
}

// PublishEventName 发布事件名称
func (r *Router) PublishEventName(eventName string, data map[string]any) {
	r.eventPublisher.PublishEventName(eventName, data)
}

// SubscribeEvent 订阅事件
func (r *Router) SubscribeEvent(event string, handler EventHandler) int {
	if r.eventBus != nil {
		return r.eventBus.Subscribe(event, handler)
	}
	return -1
}

// FireTrigger 触发触发器
func (r *Router) FireTrigger(event *Event) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.FireEvent(event)
}

// RegisterTrigger 注册触发器
func (r *Router) RegisterTrigger(trigger *Trigger) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.RegisterTrigger(trigger)
}

// GetStats 获取统计信息
func (r *Router) GetStats() *RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return &RouterStats{
		TotalCalls:      r.stats.TotalCalls,
		SuccessfulCalls: r.stats.SuccessfulCalls,
		FailedCalls:     r.stats.FailedCalls,
		PanicCalls:      r.stats.PanicCalls,
		AvgCallDuration: r.stats.AvgCallDuration,
		StartTime:       r.stats.StartTime,
		FunctionCount:   len(r.functions),
	}
}

// GetConfig 获取配置
func (r *Router) GetConfig() *RouterConfig {
	return r.config
}

// SetRecoveryHandler 设置恢复处理器
func (r *Router) SetRecoveryHandler(handler RecoveryHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryHandler = handler
}

// Shutdown 关闭路由器
func (r *Router) Shutdown() {
	r.eventPublisher.Shutdown()
}
