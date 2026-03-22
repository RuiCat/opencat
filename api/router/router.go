package router

import (
	"fmt"
	"reflect"
	"time"
)

// 路由器事件常量
const (
	// 函数管理事件
	EventFunctionRegistered   = "router.function.registered"   // 函数注册成功
	EventFunctionUnregistered = "router.function.unregistered" // 函数注销成功
	EventFunctionEnabled      = "router.function.enabled"      // 函数启用
	EventFunctionDisabled     = "router.function.disabled"     // 函数禁用

	// 拦截器事件
	EventInterceptorAdded   = "router.interceptor.added"   // 拦截器添加
	EventInterceptorRemoved = "router.interceptor.removed" // 拦截器移除
	EventInterceptorCleared = "router.interceptor.cleared" // 拦截器清空

	// 触发器事件
	EventTriggerRegistered   = "router.trigger.registered"   // 触发器注册
	EventTriggerUnregistered = "router.trigger.unregistered" // 触发器注销
	EventTriggerEnabled      = "router.trigger.enabled"      // 触发器启用
	EventTriggerDisabled     = "router.trigger.disabled"     // 触发器禁用
	EventTriggerFired        = "router.trigger.fired"        // 触发器触发
	EventTriggerError        = "router.trigger.error"        // 触发器执行错误
)

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
	if fn.CreatedAt.IsZero() {
		fn.CreatedAt = time.Now()
	}
	if fn.Stats == nil {
		fn.Stats = &FunctionStats{}
	}
	if !fn.Enabled {
		fn.Enabled = true
	}
	if fn.Function == nil {
		return fmt.Errorf("函数实现不能为空")
	}
	val := reflect.ValueOf(fn.Function)
	if val.Kind() != reflect.Func {
		return fmt.Errorf("Function 字段必须是函数类型，当前类型: %v", val.Kind())
	}
	tys := val.Type()
	if numIn := tys.NumIn(); numIn != len(fn.InputSchema) {
		firstArg := tys.In(0)
		if fn.Namespace != "" && numIn == len(fn.InputSchema)+1 && (firstArg.Kind() == reflect.Struct || (firstArg.Kind() == reflect.Ptr && firstArg.Elem().Kind() == reflect.Struct)) {
			fn.IsMethod = true
		} else {
			return fmt.Errorf("函数参数数量不匹配: 函数有 %d 个参数，InputSchema 有 %d 个参数", numIn, len(fn.InputSchema))
		}
	}
	r.functions[fn.Name] = fn
	r.stats.FunctionCount = len(r.functions)

	// 发布函数注册事件
	go r.PublishEventName(EventFunctionRegistered, map[string]any{
		"name":        fn.Name,
		"namespace":   fn.Namespace,
		"description": fn.Description,
		"builtin":     fn.Builtin,
		"is_method":   fn.IsMethod,
		"timestamp":   time.Now().UnixNano(),
	})

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

	// 发布函数注销事件
	go r.PublishEventName(EventFunctionUnregistered, map[string]any{
		"name":        name,
		"namespace":   fn.Namespace,
		"description": fn.Description,
		"builtin":     fn.Builtin,
		"timestamp":   time.Now().UnixNano(),
	})

	return nil
}

// GetFunction 获取函数信息
func (r *Router) GetFunction(name string) *Function {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.functions[name]
}

// ListFunctions 列出所有函数
func (r *Router) ListFunctions() map[string]*Function {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.functions
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
	// 发布函数启用事件
	go r.PublishEventName(EventFunctionEnabled, map[string]any{
		"name":        name,
		"namespace":   fn.Namespace,
		"description": fn.Description,
		"builtin":     fn.Builtin,
		"timestamp":   time.Now().UnixNano(),
	})
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
	// 发布函数禁用事件
	go r.PublishEventName(EventFunctionDisabled, map[string]any{
		"name":        name,
		"namespace":   fn.Namespace,
		"description": fn.Description,
		"builtin":     fn.Builtin,
		"timestamp":   time.Now().UnixNano(),
	})
	return nil
}

// AddInterceptor 添加拦截器
func (r *Router) AddInterceptor(name string, interceptor Interceptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.interceptors[name]; ok {
		return fmt.Errorf("拦截器已存在: %s", name)
	}
	r.interceptors[name] = interceptor
	// 发布拦截器添加事件
	go r.PublishEventName(EventInterceptorAdded, map[string]any{
		"name":        name,
		"interceptor": fmt.Sprintf("%T", interceptor),
		"timestamp":   time.Now().UnixNano(),
	})
	return nil
}

// RemoveInterceptor 移除拦截器
func (r *Router) RemoveInterceptor(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.interceptors[name]; !ok {
		return fmt.Errorf("拦截器不存在: %s", name)
	}
	delete(r.interceptors, name)
	// 发布拦截器移除事件
	go r.PublishEventName(EventInterceptorRemoved, map[string]any{
		"name":      name,
		"timestamp": time.Now().UnixNano(),
	})
	return nil
}

// ClearInterceptors 清空所有拦截器
func (r *Router) ClearInterceptors() {
	r.mu.Lock()
	defer r.mu.Unlock()
	interceptorCount := len(r.interceptors)
	r.interceptors = make(map[string]Interceptor, 0)
	// 发布拦截器清空事件
	go r.PublishEventName(EventInterceptorCleared, map[string]any{
		"cleared_count": interceptorCount,
		"timestamp":     time.Now().UnixNano(),
	})
}

// InterceptorCount 获取拦截器数量
func (r *Router) InterceptorCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.interceptors)
}

// PublishEvent 发布事件
func (r *Router) PublishEvent(event string, blockType BlockType, data any) {
	if r.eventBus != nil {
		r.eventBus.Publish(event, blockType, data)
	}
}

// PublishEventName 发布事件到事件总线和触发器管理器
func (r *Router) PublishEventName(eventName string, data map[string]any) {
	if r == nil {
		return
	}
	event := &Event{
		Name:   eventName,
		Source: "router",
		Data:   data,
		Time:   time.Now(),
	}
	if traceID, ok := data["trace_id"].(string); ok {
		event.TraceID = traceID
	}
	if r.eventBus != nil {
		r.eventBus.Publish(eventName, BlockTypeEvent, event)
	}
	if r.config.EnableTriggers && r.triggerManager != nil {
		go r.triggerManager.FireEvent(event)
	}
}

// SubscribeEvent 订阅事件
func (r *Router) SubscribeEvent(event string, handler EventHandler) int {
	if r.eventBus != nil {
		return r.eventBus.Subscribe(event, handler)
	}
	return -1
}

// FireTrigger 触发事件（通过触发器管理器）
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

// GetStats 获取路由器统计信息
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

// GetConfig 获取路由器配置
func (r *Router) GetConfig() *RouterConfig {
	return r.config
}

// SetRecoveryHandler 设置崩溃恢复处理器
func (r *Router) SetRecoveryHandler(handler RecoveryHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryHandler = handler
}

// NewDataBlock 创建新的数据块
func NewDataBlock(target, source string, payload map[string]any) *DataBlock {
	return &DataBlock{
		ID:        fmt.Sprintf("block-%d", time.Now().UnixMicro()),
		Type:      BlockTypeCommand,
		Timestamp: time.Now().UnixMilli(),
		Source:    source,
		Target:    target,
		Payload:   payload,
		Metadata:  make(map[string]string),
		TraceID:   fmt.Sprintf("block-%d", time.Now().UnixNano()),
	}
}

// SuccessResult 创建成功结果
func SuccessResult(data any, traceID string) *Result {
	return &Result{
		Success: true,
		Data:    data,
		TraceID: traceID,
	}
}

// ErrorResult 创建错误结果
func ErrorResult(code, message, traceID string) *Result {
	return &Result{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		TraceID: traceID,
	}
}

// RetryableErrorResult 创建可重试的错误结果
func RetryableErrorResult(code, message, traceID string) *Result {
	return &Result{
		Success: false,
		Error: &ErrorInfo{
			Code:      code,
			Message:   message,
			Retryable: true,
		},
		TraceID: traceID,
	}
}
