package router

import (
	"fmt"
	"time"
)

// BaseInterceptor 基础拦截器
type BaseInterceptor struct {
	Name string
}

// Before 执行前拦截
func (bi *BaseInterceptor) Before(ctx *Context, block *DataBlock) error {
	return nil
}

// After 执行后拦截
func (bi *BaseInterceptor) After(ctx *Context, block *DataBlock, result *Result) {
}

// LoggingInterceptor 日志拦截器
type LoggingInterceptor struct {
	BaseInterceptor
}

// NewLoggingInterceptor 创建日志拦截器
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "logging"},
	}
}

// Before 执行前记录日志
func (li *LoggingInterceptor) Before(ctx *Context, block *DataBlock) error {
	ctx.LogInfo("开始执行函数", map[string]interface{}{
		"function": block.Target,
		"source":   block.Source,
		"trace_id": block.TraceID,
		"depth":    ctx.GetCallDepth(),
	})
	return nil
}

// After 执行后记录日志
func (li *LoggingInterceptor) After(ctx *Context, block *DataBlock, result *Result) {
	fields := map[string]interface{}{
		"function": block.Target,
		"success":  result.Success,
		"duration": result.Duration,
		"trace_id": block.TraceID,
	}

	if result.Error != nil {
		fields["error_code"] = result.Error.Code
		fields["error_message"] = result.Error.Message
		ctx.LogError("函数执行失败", fields)
	} else {
		ctx.LogInfo("函数执行成功", fields)
	}
}

// TimingInterceptor 计时拦截器
type TimingInterceptor struct {
	BaseInterceptor
	startTimes map[string]time.Time
}

// NewTimingInterceptor 创建计时拦截器
func NewTimingInterceptor() *TimingInterceptor {
	return &TimingInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "timing"},
		startTimes:      make(map[string]time.Time),
	}
}

// Before 记录开始时间
func (ti *TimingInterceptor) Before(ctx *Context, block *DataBlock) error {
	ti.startTimes[block.TraceID] = time.Now()
	return nil
}

// After 计算执行时间并添加到结果
func (ti *TimingInterceptor) After(ctx *Context, block *DataBlock, result *Result) {
	if startTime, exists := ti.startTimes[block.TraceID]; exists {
		duration := time.Since(startTime)
		result.Duration = duration.Milliseconds()

		// 添加性能指标到日志
		ctx.LogDebug("函数执行时间", map[string]interface{}{
			"function":    block.Target,
			"duration_ms": duration.Milliseconds(),
			"trace_id":    block.TraceID,
		})

		delete(ti.startTimes, block.TraceID)
	}
}

// ValidationInterceptor 验证拦截器
type ValidationInterceptor struct {
	BaseInterceptor
}

// NewValidationInterceptor 创建验证拦截器
func NewValidationInterceptor() *ValidationInterceptor {
	return &ValidationInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "validation"},
	}
}

// Before 验证输入参数
func (vi *ValidationInterceptor) Before(ctx *Context, block *DataBlock) error {
	// 获取函数定义
	fn, err := ctx.Router.GetFunction(block.Target)
	if err != nil {
		return err
	}

	// 简化验证：检查必要参数是否存在
	// 实际应用中应该使用完整的 JSON Schema 验证
	if fn.InputSchema != nil {
		if required, ok := fn.InputSchema["required"].([]interface{}); ok {
			for _, req := range required {
				if param, ok := req.(string); ok {
					if _, exists := block.Payload[param]; !exists {
						return fmt.Errorf("缺少必要参数: %s", param)
					}
				}
			}
		}
	}

	return nil
}

// RateLimitInterceptor 限流拦截器
type RateLimitInterceptor struct {
	BaseInterceptor
	limits map[string]*RateLimit
}

// RateLimit 限流配置
type RateLimit struct {
	MaxRequests int
	Window      time.Duration
	requests    []time.Time
}

// NewRateLimitInterceptor 创建限流拦截器
func NewRateLimitInterceptor() *RateLimitInterceptor {
	return &RateLimitInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "rate_limit"},
		limits:          make(map[string]*RateLimit),
	}
}

// AddLimit 添加限流规则
func (rli *RateLimitInterceptor) AddLimit(function string, maxRequests int, window time.Duration) {
	rli.limits[function] = &RateLimit{
		MaxRequests: maxRequests,
		Window:      window,
		requests:    make([]time.Time, 0),
	}
}

// Before 检查限流
func (rli *RateLimitInterceptor) Before(ctx *Context, block *DataBlock) error {
	limit, exists := rli.limits[block.Target]
	if !exists {
		return nil
	}

	now := time.Now()
	windowStart := now.Add(-limit.Window)

	// 清理过期的请求记录
	validRequests := make([]time.Time, 0)
	for _, reqTime := range limit.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	limit.requests = validRequests

	// 检查是否超过限制
	if len(limit.requests) >= limit.MaxRequests {
		return fmt.Errorf("函数 %s 调用频率超过限制: %d 次/%v", block.Target, limit.MaxRequests, limit.Window)
	}

	// 记录本次请求
	limit.requests = append(limit.requests, now)
	return nil
}

// AuthInterceptor 认证拦截器
type AuthInterceptor struct {
	BaseInterceptor
	allowedAgents map[string][]string // agentID -> 允许的函数列表
}

// NewAuthInterceptor 创建认证拦截器
func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "auth"},
		allowedAgents:   make(map[string][]string),
	}
}

// Allow 允许特定智能体调用特定函数
func (ai *AuthInterceptor) Allow(agentID string, functions ...string) {
	if _, exists := ai.allowedAgents[agentID]; !exists {
		ai.allowedAgents[agentID] = make([]string, 0)
	}
	ai.allowedAgents[agentID] = append(ai.allowedAgents[agentID], functions...)
}

// Before 检查权限
func (ai *AuthInterceptor) Before(ctx *Context, block *DataBlock) error {
	// 如果智能体不在允许列表中，检查是否有权限
	if allowedFuncs, exists := ai.allowedAgents[ctx.AgentID]; exists {
		// 检查是否允许调用该函数
		for _, funcName := range allowedFuncs {
			if funcName == block.Target || funcName == "*" {
				return nil
			}
		}
	}

	// 默认情况下，允许调用 router.* 和 sys.* 函数（系统函数）
	// 这是简化实现，实际应用中应该有更复杂的权限模型
	if len(block.Target) >= 7 && (block.Target[:7] == "router." || block.Target[:4] == "sys." || block.Target[:5] == "util.") {
		return nil
	}

	return fmt.Errorf("智能体 %s 没有权限调用函数 %s", ctx.AgentID, block.Target)
}

// MetricsInterceptor 指标拦截器
type MetricsInterceptor struct {
	BaseInterceptor
	metrics map[string]*FunctionMetrics
}

// FunctionMetrics 函数指标
type FunctionMetrics struct {
	CallCount     int64
	SuccessCount  int64
	ErrorCount    int64
	TotalDuration time.Duration
	LastCallTime  time.Time
}

// NewMetricsInterceptor 创建指标拦截器
func NewMetricsInterceptor() *MetricsInterceptor {
	return &MetricsInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "metrics"},
		metrics:         make(map[string]*FunctionMetrics),
	}
}

// Before 记录调用开始
func (mi *MetricsInterceptor) Before(ctx *Context, block *DataBlock) error {
	if _, exists := mi.metrics[block.Target]; !exists {
		mi.metrics[block.Target] = &FunctionMetrics{}
	}
	mi.metrics[block.Target].CallCount++
	return nil
}

// After 记录调用结果
func (mi *MetricsInterceptor) After(ctx *Context, block *DataBlock, result *Result) {
	metrics := mi.metrics[block.Target]
	metrics.LastCallTime = time.Now()

	if result.Success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}

	if result.Duration > 0 {
		metrics.TotalDuration += time.Duration(result.Duration) * time.Millisecond
	}
}

// GetMetrics 获取指标
func (mi *MetricsInterceptor) GetMetrics(function string) *FunctionMetrics {
	return mi.metrics[function]
}

// GetAllMetrics 获取所有指标
func (mi *MetricsInterceptor) GetAllMetrics() map[string]*FunctionMetrics {
	return mi.metrics
}

// CircuitBreakerInterceptor 熔断器拦截器
type CircuitBreakerInterceptor struct {
	BaseInterceptor
	breakers map[string]*CircuitBreaker
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	FailureThreshold int           // 失败阈值
	ResetTimeout     time.Duration // 重置超时
	State            string        // closed, open, half-open
	FailureCount     int
	LastFailureTime  time.Time
}

// NewCircuitBreakerInterceptor 创建熔断器拦截器
func NewCircuitBreakerInterceptor() *CircuitBreakerInterceptor {
	return &CircuitBreakerInterceptor{
		BaseInterceptor: BaseInterceptor{Name: "circuit_breaker"},
		breakers:        make(map[string]*CircuitBreaker),
	}
}

// AddBreaker 添加熔断器
func (cbi *CircuitBreakerInterceptor) AddBreaker(function string, failureThreshold int, resetTimeout time.Duration) {
	cbi.breakers[function] = &CircuitBreaker{
		FailureThreshold: failureThreshold,
		ResetTimeout:     resetTimeout,
		State:            "closed",
		FailureCount:     0,
	}
}

// Before 检查熔断器状态
func (cbi *CircuitBreakerInterceptor) Before(ctx *Context, block *DataBlock) error {
	breaker, exists := cbi.breakers[block.Target]
	if !exists {
		return nil
	}

	now := time.Now()

	switch breaker.State {
	case "open":
		// 检查是否应该进入半开状态
		if now.Sub(breaker.LastFailureTime) > breaker.ResetTimeout {
			breaker.State = "half-open"
			breaker.FailureCount = 0
			return nil
		}
		return fmt.Errorf("函数 %s 已熔断，请稍后重试", block.Target)

	case "half-open":
		// 允许一次尝试
		return nil

	case "closed":
		return nil
	}

	return nil
}

// After 更新熔断器状态
func (cbi *CircuitBreakerInterceptor) After(ctx *Context, block *DataBlock, result *Result) {
	breaker, exists := cbi.breakers[block.Target]
	if !exists {
		return
	}

	if !result.Success {
		breaker.FailureCount++
		breaker.LastFailureTime = time.Now()

		if breaker.State == "half-open" {
			// 半开状态下失败，重新打开
			breaker.State = "open"
		} else if breaker.FailureCount >= breaker.FailureThreshold {
			// 达到失败阈值，打开熔断器
			breaker.State = "open"
		}
	} else {
		if breaker.State == "half-open" {
			// 半开状态下成功，关闭熔断器
			breaker.State = "closed"
			breaker.FailureCount = 0
		} else {
			// 成功调用，重置失败计数
			breaker.FailureCount = 0
		}
	}
}
