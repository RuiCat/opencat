package router

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// RouterConfig 路由配置
type RouterConfig struct {
	MaxCallDepth       int           // 最大调用深度
	DefaultTimeout     time.Duration // 默认超时时间
	EnableAuditLog     bool          // 是否启用审计日志
	EnableRecovery     bool          // 是否启用崩溃恢复
	MaxFunctions       int           // 最大函数数量
	MaxConcurrentCalls int           // 最大并发调用数

	// 触发器配置
	EnableTriggers bool                  // 是否启用触发器
	TriggerConfig  *TriggerManagerConfig // 触发器管理器配置
}

// DefaultConfig 默认配置
func DefaultConfig() *RouterConfig {
	return &RouterConfig{
		MaxCallDepth:       10,
		DefaultTimeout:     30 * time.Second,
		EnableAuditLog:     true,
		EnableRecovery:     true,
		MaxFunctions:       1000,
		MaxConcurrentCalls: 100,
		EnableTriggers:     true,
		TriggerConfig:      DefaultTriggerManagerConfig(),
	}
}

// Router 路由执行器
type Router struct {
	mu              sync.RWMutex
	functions       map[string]*Function // 函数注册表
	interceptors    []Interceptor        // 拦截器链
	logger          *AuditLogger         // 审计日志
	recoveryHandler RecoveryHandler      // 全局崩溃处理
	eventBus        *EventBus            // 事件总线
	config          *RouterConfig        // 配置
	callSemaphore   chan struct{}        // 并发控制信号量
	stats           *RouterStats         // 路由统计

	// 触发器管理器
	triggerManager *TriggerManager // 触发器管理器
}

// RouterStats 路由统计
type RouterStats struct {
	TotalCalls      int64         `json:"total_calls"`
	SuccessfulCalls int64         `json:"successful_calls"`
	FailedCalls     int64         `json:"failed_calls"`
	PanicCalls      int64         `json:"panic_calls"`
	AvgCallDuration time.Duration `json:"avg_call_duration"`
	StartTime       time.Time     `json:"start_time"`
	FunctionCount   int           `json:"function_count"`
}

// NewRouter 创建新的路由
func NewRouter(config *RouterConfig) *Router {
	if config == nil {
		config = DefaultConfig()
	}

	router := &Router{
		functions:       make(map[string]*Function),
		interceptors:    make([]Interceptor, 0),
		logger:          NewAuditLogger(),
		recoveryHandler: defaultRecoveryHandler,
		eventBus:        NewEventBus(),
		config:          config,
		callSemaphore:   make(chan struct{}, config.MaxConcurrentCalls),
		stats: &RouterStats{
			StartTime: time.Now(),
		},
	}

	// 初始化触发器管理器
	if config.EnableTriggers {
		router.triggerManager = NewTriggerManager(router.eventBus, config.TriggerConfig)
	}

	return router
}

// Register 注册函数
func (r *Router) Register(fn *Function) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查函数数量限制
	if len(r.functions) >= r.config.MaxFunctions {
		return fmt.Errorf("达到最大函数数量限制: %d", r.config.MaxFunctions)
	}

	// 检查是否已存在
	if _, exists := r.functions[fn.Name]; exists {
		return fmt.Errorf("函数已存在: %s", fn.Name)
	}

	// 设置默认值
	if fn.CreatedAt.IsZero() {
		fn.CreatedAt = time.Now()
	}
	if fn.Stats == nil {
		fn.Stats = &FunctionStats{}
	}
	if !fn.Enabled {
		fn.Enabled = true
	}

	// 注册函数
	r.functions[fn.Name] = fn
	r.stats.FunctionCount = len(r.functions)

	// 发布事件
	r.eventBus.Publish("function.registered", map[string]interface{}{
		"name":      fn.Name,
		"namespace": fn.Namespace,
		"builtin":   fn.Builtin,
	})

	return nil
}

// Unregister 注销函数
func (r *Router) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}

	// 检查是否为内置函数
	if fn.Builtin {
		return fmt.Errorf("不能注销内置函数: %s", name)
	}

	delete(r.functions, name)
	r.stats.FunctionCount = len(r.functions)

	// 发布事件
	r.eventBus.Publish("function.unregistered", map[string]interface{}{
		"name":      name,
		"namespace": fn.Namespace,
	})

	return nil
}

// Call 调用函数
func (r *Router) Call(ctx *Context, block *DataBlock) *Result {
	startTime := time.Now()

	// 发布函数调用开始事件
	r.publishEvent(EventFunctionCalled, map[string]interface{}{
		"function":  block.Target,
		"caller":    ctx.AgentID,
		"trace_id":  block.TraceID,
		"depth":     ctx.GetCallDepth(),
		"timestamp": startTime.UnixNano(),
	})

	// 检查调用深度
	if ctx.GetCallDepth() >= r.config.MaxCallDepth {
		result := &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "MAX_CALL_DEPTH_EXCEEDED",
				Message: fmt.Sprintf("调用深度超过限制: %d", r.config.MaxCallDepth),
			},
			Duration: time.Since(startTime).Milliseconds(),
			TraceID:  block.TraceID,
		}

		// 发布调用失败事件
		r.publishEvent(EventFunctionFailed, map[string]interface{}{
			"function": block.Target,
			"error":    result.Error,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"reason":   "max_call_depth_exceeded",
		})

		return result
	}

	// 获取并发许可
	select {
	case r.callSemaphore <- struct{}{}:
		defer func() { <-r.callSemaphore }()
	case <-time.After(100 * time.Millisecond):
		result := &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:      "CONCURRENT_LIMIT_EXCEEDED",
				Message:   "并发调用数达到限制",
				Retryable: true,
			},
			Duration: time.Since(startTime).Milliseconds(),
			TraceID:  block.TraceID,
		}

		// 发布调用失败事件
		r.publishEvent(EventFunctionFailed, map[string]interface{}{
			"function": block.Target,
			"error":    result.Error,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"reason":   "concurrent_limit_exceeded",
		})

		return result
	}

	// 查找函数
	r.mu.RLock()
	fn, exists := r.functions[block.Target]
	r.mu.RUnlock()

	if !exists {
		result := &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "FUNCTION_NOT_FOUND",
				Message: fmt.Sprintf("函数不存在: %s", block.Target),
			},
			Duration: time.Since(startTime).Milliseconds(),
			TraceID:  block.TraceID,
		}

		// 发布调用失败事件
		r.publishEvent(EventFunctionFailed, map[string]interface{}{
			"function": block.Target,
			"error":    result.Error,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"reason":   "function_not_found",
		})

		return result
	}

	// 检查函数是否启用
	if !fn.Enabled {
		result := &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "FUNCTION_DISABLED",
				Message: fmt.Sprintf("函数已禁用: %s", block.Target),
			},
			Duration: time.Since(startTime).Milliseconds(),
			TraceID:  block.TraceID,
		}

		// 发布调用失败事件
		r.publishEvent(EventFunctionFailed, map[string]interface{}{
			"function": block.Target,
			"error":    result.Error,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"reason":   "function_disabled",
		})

		return result
	}

	// 执行拦截器链（Before）
	for _, interceptor := range r.interceptors {
		if err := interceptor.Before(ctx, block); err != nil {
			result := &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "INTERCEPTOR_ERROR",
					Message: err.Error(),
				},
				Duration: time.Since(startTime).Milliseconds(),
				TraceID:  block.TraceID,
			}

			// 发布调用失败事件
			r.publishEvent(EventFunctionFailed, map[string]interface{}{
				"function": block.Target,
				"error":    result.Error,
				"trace_id": block.TraceID,
				"duration": result.Duration,
				"reason":   "interceptor_error",
			})

			return result
		}
	}

	// 执行函数
	var result *Result
	if r.config.EnableRecovery {
		result = r.callWithRecovery(ctx, block, fn)
	} else {
		result = r.callDirect(ctx, block, fn)
	}

	// 更新统计
	r.updateStats(result, fn, time.Since(startTime))

	// 记录审计日志
	if r.config.EnableAuditLog {
		var err error
		if result.Error != nil {
			err = fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
		}
		r.logger.Log("function.call", ctx.AgentID, block.Target, result.Success, time.Since(startTime), err, map[string]interface{}{
			"trace_id": block.TraceID,
			"depth":    ctx.GetCallDepth(),
		})
	}

	// 执行拦截器链（After）
	for _, interceptor := range r.interceptors {
		interceptor.After(ctx, block, result)
	}

	// 设置执行时间
	result.Duration = time.Since(startTime).Milliseconds()

	// 发布调用结果事件
	if result.Success {
		r.publishEvent(EventFunctionSuccess, map[string]interface{}{
			"function": block.Target,
			"result":   result.Data,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"caller":   ctx.AgentID,
		})
	} else {
		r.publishEvent(EventFunctionFailed, map[string]interface{}{
			"function": block.Target,
			"error":    result.Error,
			"trace_id": block.TraceID,
			"duration": result.Duration,
			"caller":   ctx.AgentID,
			"reason":   "execution_failed",
		})
	}

	return result
}

// callDirect 直接调用函数
func (r *Router) callDirect(ctx *Context, block *DataBlock, fn *Function) *Result {
	// 更新函数统计
	fn.Stats.CallCount++
	fn.Stats.LastCalledAt = time.Now()

	// 执行函数
	result := fn.Handler(ctx, block)

	// 更新成功/失败统计
	if result.Success {
		fn.Stats.SuccessCount++
	} else {
		fn.Stats.FailureCount++
	}

	return result
}

// callWithRecovery 带崩溃恢复的调用
func (r *Router) callWithRecovery(ctx *Context, block *DataBlock, fn *Function) *Result {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			// 更新panic统计
			fn.Stats.PanicCount++
			r.stats.PanicCalls++

			// 调用恢复处理器
			if r.recoveryHandler != nil {
				recoveryResult := r.recoveryHandler(ctx, block, panicValue)
				if recoveryResult != nil {
					// 使用恢复结果
					recoveryResult.Duration = time.Now().UnixNano()/1e6 - block.Timestamp/1e6
					recoveryResult.TraceID = block.TraceID
					// 注意：这里不能直接返回，因为我们在defer中
					// 实际应该通过channel或其他机制传递结果
				}
			}

			// 记录panic日志
			ctx.LogError("函数执行panic", map[string]interface{}{
				"function": fn.Name,
				"panic":    panicValue,
				"stack":    string(debug.Stack()),
			})
		}
	}()

	return r.callDirect(ctx, block, fn)
}

// updateStats 更新统计信息
func (r *Router) updateStats(result *Result, fn *Function, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.TotalCalls++
	if result.Success {
		r.stats.SuccessfulCalls++
	} else {
		r.stats.FailedCalls++
	}

	// 更新平均调用时长
	if r.stats.TotalCalls == 1 {
		r.stats.AvgCallDuration = duration
	} else {
		// 加权平均
		r.stats.AvgCallDuration = (r.stats.AvgCallDuration*time.Duration(r.stats.TotalCalls-1) + duration) / time.Duration(r.stats.TotalCalls)
	}

	// 更新函数平均调用时长
	if fn.Stats.CallCount == 1 {
		fn.Stats.AvgDuration = duration
	} else {
		fn.Stats.AvgDuration = (fn.Stats.AvgDuration*time.Duration(fn.Stats.CallCount-1) + duration) / time.Duration(fn.Stats.CallCount)
	}
}

// Use 添加拦截器
func (r *Router) Use(interceptor Interceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interceptors = append(r.interceptors, interceptor)
}

// List 列出函数
func (r *Router) List(namespace string) []*Function {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Function, 0)
	for _, fn := range r.functions {
		if namespace == "" || fn.Namespace == namespace {
			result = append(result, fn)
		}
	}

	return result
}

// Enable 启用函数
func (r *Router) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}

	fn.Enabled = true
	return nil
}

// Disable 禁用函数
func (r *Router) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fn, exists := r.functions[name]
	if !exists {
		return fmt.Errorf("函数不存在: %s", name)
	}

	// 不能禁用内置函数
	if fn.Builtin {
		return fmt.Errorf("不能禁用内置函数: %s", name)
	}

	fn.Enabled = false
	return nil
}

// GetStats 获取路由统计
func (r *Router) GetStats() *RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := *r.stats
	stats.FunctionCount = len(r.functions)
	return &stats
}

// GetFunctionStats 获取函数统计
func (r *Router) GetFunctionStats(name string) (*FunctionStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, exists := r.functions[name]
	if !exists {
		return nil, fmt.Errorf("函数不存在: %s", name)
	}

	return fn.Stats, nil
}

// QueryLogs 查询审计日志
func (r *Router) QueryLogs(filter AuditLogFilter) []AuditLog {
	return r.logger.Query(filter)
}

// SetRecoveryHandler 设置崩溃恢复处理器
func (r *Router) SetRecoveryHandler(handler RecoveryHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryHandler = handler
}

// Subscribe 订阅事件
func (r *Router) Subscribe(event string, handler EventHandler) {
	r.eventBus.Subscribe(event, handler)
}

// Publish 发布事件
func (r *Router) Publish(event string, data interface{}) {
	r.eventBus.Publish(event, data)
}

// GetFunction 获取函数
func (r *Router) GetFunction(name string) (*Function, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, exists := r.functions[name]
	if !exists {
		return nil, fmt.Errorf("函数不存在: %s", name)
	}

	return fn, nil
}

// defaultRecoveryHandler 默认崩溃恢复处理器
func defaultRecoveryHandler(ctx *Context, block *DataBlock, panicValue interface{}) *Result {
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
}

// publishEvent 发布事件（内部方法）
func (r *Router) publishEvent(eventName string, data map[string]interface{}) {
	// 创建事件对象
	event := NewEvent(eventName, "router", data)

	// 发布到事件总线
	r.eventBus.Publish(eventName, event)

	// 如果启用了触发器管理器，处理事件
	if r.config.EnableTriggers && r.triggerManager != nil {
		go func() {
			if err := r.triggerManager.HandleEvent(event); err != nil {
				// 记录错误但不阻塞主流程
				fmt.Printf("[触发器处理错误] %s: %v\n", eventName, err)
			}
		}()
	}
}

// ============================================================================
// 触发器管理相关方法
// ============================================================================

// GetTriggerManager 获取触发器管理器
func (r *Router) GetTriggerManager() *TriggerManager {
	return r.triggerManager
}

// RegisterTrigger 注册触发器
func (r *Router) RegisterTrigger(trigger *Trigger) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.Register(trigger)
}

// UnregisterTrigger 注销触发器
func (r *Router) UnregisterTrigger(id string) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.Unregister(id)
}

// ListTriggers 列出所有触发器
func (r *Router) ListTriggers() []*Trigger {
	if r.triggerManager == nil {
		return []*Trigger{}
	}
	return r.triggerManager.List()
}

// EnableTrigger 启用触发器
func (r *Router) EnableTrigger(id string) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.Enable(id)
}

// DisableTrigger 禁用触发器
func (r *Router) DisableTrigger(id string) error {
	if r.triggerManager == nil {
		return fmt.Errorf("触发器管理器未启用")
	}
	return r.triggerManager.Disable(id)
}

// PublishCustomEvent 发布自定义事件
func (r *Router) PublishCustomEvent(eventName string, data map[string]interface{}) error {
	if !r.config.EnableTriggers {
		return fmt.Errorf("触发器功能未启用")
	}

	event := NewEvent(eventName, "custom", data)

	// 发布到事件总线
	r.eventBus.Publish(eventName, event)

	// 处理事件
	if r.triggerManager != nil {
		return r.triggerManager.HandleEvent(event)
	}

	return nil
}
