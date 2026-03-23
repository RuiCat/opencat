// 上下文管理包
package router

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// FunctionWrapper 函数包装器
type FunctionWrapper struct {
	original any               // 原始函数
	chain    *InterceptorChain // 函数特定的拦截器链
	metadata *FunctionMetadata // 函数元数据
}

// FunctionMetadata 函数元数据
type FunctionMetadata struct {
	Name string       // 函数名称
	Type reflect.Type // 函数类型
}

// Context 执行上下文
type Context struct {
	*Router
	SessionID       string             // 会话唯一标识符
	AgentID         string             // 调用者身份标识
	TraceID         string             // 分布式追踪ID
	StartTime       time.Time          // 上下文创建时间
	Values          map[string]any     // 上下文键值对存储
	mu              sync.RWMutex       // 读写互斥锁
	Timeout         time.Duration      // 执行超时时间
	cancel          context.CancelFunc // 取消函数
	CallDepth       int                // 当前调用深度
	ParentID        string             // 父调用ID
	wrappers        sync.Map           // 函数包装器映射
	interceptor     *InterceptorChain  // 全局拦截器链
	recoveryHandler RecoveryHandler    // 错误恢复
}

// SetRecoveryHandler 设置恢复处理器
func (c *Context) SetRecoveryHandler(handler RecoveryHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recoveryHandler = handler
}

// GetCallDepth 获取当前调用深度
func (c *Context) GetCallDepth() int {
	return c.CallDepth
}

// IncrementCallDepth 增加调用深度
func (c *Context) IncrementCallDepth() {
	c.CallDepth++
}

// DecrementCallDepth 减少调用深度
func (c *Context) DecrementCallDepth() {
	if c.CallDepth > 0 {
		c.CallDepth--
	}
}

// SetValue 设置上下文键值对
// key: 键名
// value: 值
func (c *Context) SetValue(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Values == nil {
		c.Values = make(map[string]any)
	}
	c.Values[key] = value
}

// GetValue 获取上下文值
// key: 键名
// 返回: 对应的值
func (c *Context) GetValue(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Values == nil {
		return nil
	}
	return c.Values[key]
}

// DeleteValue 删除上下文键值对
// key: 要删除的键名
func (c *Context) DeleteValue(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Values != nil {
		delete(c.Values, key)
	}
}

// LogError 记录错误日志
// message: 错误消息
// fields: 附加字段
func (c *Context) LogError(message string, fields map[string]any) {
	fmt.Printf("[ERROR] %s: %v\n", message, fields)
}

// LogErrorEvent 记录错误事件
func (c *Context) LogErrorEvent(eventName string, message string, data map[string]any) {
	fields := map[string]any{
		"caller":    c.AgentID,
		"trace_id":  c.TraceID,
		"timestamp": time.Now().UnixNano(),
	}
	maps.Copy(fields, data)
	c.Router.PublishEventName(eventName, fields)
	c.LogError(message, fields)
}

// RegisterWithInterceptors 注册函数并附加拦截器
func (r *Context) RegisterWithInterceptors(name, description, namespace string, inputSchema, outputSchema []string, fn any, interceptors ...Interceptor) error {
	if err := r.Register(name, description, namespace, inputSchema, outputSchema, fn); err != nil {
		return err
	}
	wrapper := &FunctionWrapper{
		original: fn,
		chain:    NewInterceptorChain(),
		metadata: &FunctionMetadata{
			Name: name,
			Type: reflect.TypeOf(fn),
		},
	}
	for i, interceptor := range interceptors {
		wrapper.chain.Add(interceptor, i*10)
	}
	r.wrappers.Store(name, wrapper)
	return nil
}

// AddGlobalInterceptor 添加全局拦截器
func (r *Context) AddGlobalInterceptor(interceptor Interceptor, priority int) {
	r.interceptor.Add(interceptor, priority)
}

// RemoveGlobalInterceptor 移除全局拦截器
func (r *Context) RemoveGlobalInterceptor(interceptor Interceptor) {
	r.interceptor.Remove(interceptor)
}

// AddFunctionInterceptor 为特定函数添加拦截器
func (r *Context) AddFunctionInterceptor(functionName string, interceptor Interceptor, priority int) error {
	wrapper, ok := r.wrappers.Load(functionName)
	if !ok {
		return fmt.Errorf("函数未找到或未使用拦截器注册: %s", functionName)
	}
	wrapper.(*FunctionWrapper).chain.Add(interceptor, priority)
	return nil
}

// RemoveFunctionInterceptor 从特定函数移除拦截器
func (r *Context) RemoveFunctionInterceptor(functionName string, interceptor Interceptor) error {
	wrapper, ok := r.wrappers.Load(functionName)
	if !ok {
		return fmt.Errorf("函数未找到或未使用拦截器注册: %s", functionName)
	}
	wrapper.(*FunctionWrapper).chain.Remove(interceptor)
	return nil
}

// Call 调用函数
func (c *Context) Call(name string, args ...any) (result any, err error) {
	// 获取函数定义
	fn := c.GetFunction(name)
	if fn == nil {
		return nil, fmt.Errorf("函数未找到: %s", name)
	}
	// 检查函数是否启用
	if !fn.Enabled {
		return nil, fmt.Errorf("函数已禁用: %s", name)
	}
	// 检查调用深度限制
	if c.GetCallDepth() >= c.GetConfig().MaxCallDepth {
		return nil, fmt.Errorf("达到最大调用深度限制: %d", c.GetConfig().MaxCallDepth)
	}
	// 增加调用深度
	c.IncrementCallDepth()
	defer c.DecrementCallDepth()
	// 发布函数调用开始事件
	startTime := time.Now()
	c.PublishEventName(EventFunctionCalled, map[string]any{
		"function":  name,
		"caller":    c.AgentID,
		"trace_id":  c.TraceID,
		"depth":     c.GetCallDepth(),
		"timestamp": startTime.UnixNano(),
	})
	// 获取函数包装器（检查是否使用拦截器注册）
	wrapper, hasWrapper := c.wrappers.Load(name)
	// 创建拦截器调用记录
	call := NewInterceptorCall(name, args)
	// 发布拦截器开始执行事件
	c.PublishEventName(EventFunctionInterceptorStart, map[string]any{
		"function":  name,
		"trace_id":  c.TraceID,
		"caller":    c.AgentID,
		"timestamp": time.Now().UnixNano(),
	})
	// 执行全局拦截器链
	if err := c.interceptor.Intercept(c, call); err != nil {
		// 发布拦截器错误事件
		c.PublishEventName(EventFunctionInterceptorError, map[string]any{
			"function":  name,
			"trace_id":  c.TraceID,
			"caller":    c.AgentID,
			"error":     err.Error(),
			"timestamp": time.Now().UnixNano(),
		})
		return nil, err
	}
	// 执行函数特定拦截器链
	if hasWrapper {
		wrapperObj := wrapper.(*FunctionWrapper)
		if err := wrapperObj.chain.Intercept(c, call); err != nil {
			// 发布拦截器错误事件
			c.PublishEventName(EventFunctionInterceptorError, map[string]any{
				"function":  name,
				"trace_id":  c.TraceID,
				"caller":    c.AgentID,
				"error":     err.Error(),
				"timestamp": time.Now().UnixNano(),
			})
			return nil, err
		}
	}
	// 发布拦截器执行结束事件
	c.PublishEventName(EventFunctionInterceptorEnd, map[string]any{
		"function":  name,
		"trace_id":  c.TraceID,
		"caller":    c.AgentID,
		"timestamp": time.Now().UnixNano(),
	})
	// 如果拦截器已经设置了结果，直接返回
	if call.Result != nil || call.Err != nil {
		return call.Result, call.Err
	}
	// 获取并发信号量（如果启用）
	if c.config.MaxConcurrentCalls > 0 {
		select {
		case c.callSemaphore <- struct{}{}:
			defer func() { <-c.callSemaphore }()
		default:
			// 发布并发限制触发事件
			c.PublishEventName(EventFunctionConcurrentLimit, map[string]any{
				"function":  name,
				"trace_id":  c.TraceID,
				"caller":    c.AgentID,
				"limit":     c.config.MaxConcurrentCalls,
				"timestamp": time.Now().UnixNano(),
			})
			return nil, fmt.Errorf("达到最大并发调用限制: %d", c.config.MaxConcurrentCalls)
		}
	}
	// 创建带超时的上下文
	var cancel context.CancelFunc
	timeout := c.config.DefaultTimeout
	if timeout > 0 {
		_, cancel = context.WithTimeout(context.Background(), timeout)
		defer func() {
			if cancel != nil {
				cancel()
			}
		}()
		// 设置取消函数到上下文
		c.SetValue("cancel", cancel)
	}
	// 执行函数调用
	defer func() {
		duration := time.Since(startTime)
		// 处理panic恢复
		panicValue := recover()
		if panicValue != nil {
			// 获取堆栈信息
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			stackTrace := "无法获取堆栈信息"
			if n > 0 {
				stackTrace = string(buf[:n])
			}
			// 调用恢复处理器（如果启用）
			if c.recoveryHandler != nil && c.config.EnableRecovery {
				block := &DataBlock{
					ID:        fmt.Sprintf("trace-%d", time.Now().UnixNano()),
					Type:      BlockTypeError,
					Timestamp: time.Now().UnixMilli(),
					Source:    c.AgentID,
					Target:    name,
					TraceID:   c.TraceID,
					Payload: map[string]any{
						"panic":     panicValue,
						"stack":     stackTrace,
						"function":  name,
						"args":      args,
						"recovered": true,
					},
				}
				recoveryResult := c.recoveryHandler(c, block, panicValue)
				if recoveryResult != nil {
					result = recoveryResult.Data
					err = fmt.Errorf("panic recovered: %v", panicValue)
				} else {
					err = fmt.Errorf("panic: %v", panicValue)
				}
			} else {
				err = fmt.Errorf("panic: %v", panicValue)
			}
			// 发布panic事件
			c.PublishEventName(EventFunctionPanic, map[string]any{
				"function":  name,
				"panic":     panicValue,
				"trace_id":  c.TraceID,
				"duration":  duration.Milliseconds(),
				"caller":    c.AgentID,
				"stack":     stackTrace,
				"recovered": true,
				"timestamp": time.Now().UnixNano(),
			})
		}
		// 完成拦截器调用记录
		call.Complete(result, err)
		// 发布函数调用事件
		eventData := map[string]any{
			"function":  name,
			"trace_id":  c.TraceID,
			"duration":  duration.Milliseconds(),
			"caller":    c.AgentID,
			"depth":     c.GetCallDepth(),
			"timestamp": time.Now().UnixNano(),
		}
		if err != nil {
			eventData["error"] = err.Error()
			eventData["success"] = false
			c.PublishEventName(EventFunctionFailed, eventData)
		} else {
			eventData["success"] = true
			c.PublishEventName(EventFunctionSuccess, eventData)
		}
		// 更新统计信息
		c.mu.Lock()
		defer c.mu.Unlock()
		if fn != nil && fn.Stats != nil {
			stats := fn.Stats
			stats.CallCount++
			stats.LastCalledAt = time.Now()
			stats.AvgDuration = (stats.AvgDuration*time.Duration(stats.CallCount-1) + duration) / time.Duration(stats.CallCount)
			if panicValue == nil {
				stats.SuccessCount++
			} else {
				stats.FailureCount++
			}
		}
		// 更新路由器全局统计
		c.stats.TotalCalls++
		if err == nil {
			c.stats.SuccessfulCalls++
		} else {
			c.stats.FailedCalls++
		}
		c.stats.AvgCallDuration = (c.stats.AvgCallDuration*time.Duration(c.stats.TotalCalls-1) + duration) / time.Duration(c.stats.TotalCalls)
	}()
	// 准备反射参数并调用函数
	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		reflectArgs[i] = reflect.ValueOf(arg)
	}
	// 获取函数值
	fnValue := reflect.ValueOf(fn.Function)
	if !fnValue.IsValid() {
		return nil, fmt.Errorf("函数无效")
	}
	// 处理方法调用
	if fn.IsMethod {
		namespaceValue := c.GetValue(fn.Namespace)
		if namespaceValue == nil {
			return nil, fmt.Errorf("命名空间未找到: %s", fn.Namespace)
		}
		reflectArgs = append([]reflect.Value{reflect.ValueOf(namespaceValue)}, reflectArgs...)
	}
	// 调用函数
	results := fnValue.Call(reflectArgs)
	// 解析结果
	if len(results) == 0 {
		return nil, nil
	}
	result = results[0].Interface()
	if len(results) > 1 {
		if errVal := results[len(results)-1].Interface(); errVal != nil {
			if e, ok := errVal.(error); ok {
				err = e
			}
		}
	}
	return result, err
}

// CallFunc 绑定函数到变量
// name: 函数名称
// call: 指向函数变量的指针，用于接收绑定的函数
// 返回: 错误信息
func (c *Context) CallFunc(name string, call any) (err error) {
	startTime := time.Now()
	// 发布函数绑定开始事件
	c.PublishEventName(EventFunctionCalled, map[string]any{
		"function":  name,
		"action":    "bind",
		"caller":    c.AgentID,
		"trace_id":  c.TraceID,
		"depth":     c.GetCallDepth(),
		"timestamp": startTime.UnixNano(),
	})
	// 获取函数定义
	fn := c.GetFunction(name)
	if fn == nil {
		errMsg := fmt.Sprintf("函数未找到: %s", name)
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"function": name,
			"error":    errMsg,
			"trace_id": c.TraceID,
			"duration": time.Since(startTime).Milliseconds(),
			"reason":   "function_not_found",
			"action":   "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	// 检查函数是否启用
	if !fn.Enabled {
		errMsg := fmt.Sprintf("函数已禁用: %s", name)
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"function": name,
			"error":    errMsg,
			"trace_id": c.TraceID,
			"duration": time.Since(startTime).Milliseconds(),
			"reason":   "function_disabled",
			"action":   "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	// 检查调用深度限制
	if c.GetCallDepth() >= c.GetConfig().MaxCallDepth {
		errMsg := fmt.Sprintf("达到最大调用深度限制: %d", c.GetConfig().MaxCallDepth)
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"function": name,
			"depth":    c.GetCallDepth(),
			"error":    errMsg,
			"trace_id": c.TraceID,
			"duration": time.Since(startTime).Milliseconds(),
			"reason":   "max_depth_exceeded",
			"action":   "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	// 验证目标变量
	targetValue := reflect.ValueOf(call)
	if targetValue.Kind() != reflect.Ptr {
		errMsg := "call 参数必须是指向函数变量的指针"
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"error":     errMsg,
			"function":  name,
			"call_type": targetValue.Kind().String(),
			"trace_id":  c.TraceID,
			"duration":  time.Since(startTime).Milliseconds(),
			"reason":    "invalid_call_type",
			"action":    "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	targetElem := targetValue.Elem()
	if targetElem.Kind() != reflect.Func {
		errMsg := "目标变量必须是函数类型"
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"function":  name,
			"elem_type": targetElem.Kind().String(),
			"error":     errMsg,
			"trace_id":  c.TraceID,
			"duration":  time.Since(startTime).Milliseconds(),
			"reason":    "invalid_target_type",
			"action":    "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	// 获取原始函数
	originalFn := reflect.ValueOf(fn.Function)
	if originalFn.Kind() != reflect.Func {
		errMsg := fmt.Sprintf("注册的函数无效: %s", name)
		c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
			"function": name,
			"error":    errMsg,
			"trace_id": c.TraceID,
			"duration": time.Since(startTime).Milliseconds(),
			"reason":   "invalid_function",
			"action":   "bind",
		})
		return fmt.Errorf("%s", errMsg)
	}
	// 处理方法绑定
	if fn.IsMethod {
		namespaceValue := c.GetValue(fn.Namespace)
		if namespaceValue == nil {
			errMsg := fmt.Sprintf("命名空间未找到: %s", fn.Namespace)
			c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
				"function":  name,
				"namespace": fn.Namespace,
				"error":     errMsg,
				"trace_id":  c.TraceID,
				"duration":  time.Since(startTime).Milliseconds(),
				"reason":    "namespace_not_found",
				"action":    "bind",
			})
			return fmt.Errorf("%s", errMsg)
		}
		// 创建方法包装器
		wrappedFn := reflect.MakeFunc(targetElem.Type(), func(args []reflect.Value) []reflect.Value {
			// 在实际调用时发布成功事件
			c.PublishEventName(EventFunctionSuccess, map[string]any{
				"function":  name,
				"namespace": fn.Namespace,
				"is_method": true,
				"action":    "execute",
				"trace_id":  c.TraceID,
				"caller":    c.AgentID,
			})
			return originalFn.Call(append([]reflect.Value{reflect.ValueOf(namespaceValue)}, args...))
		})
		targetElem.Set(wrappedFn)
	} else {
		// 直接设置函数
		// 检查类型兼容性
		if !originalFn.Type().AssignableTo(targetElem.Type()) {
			errMsg := fmt.Sprintf("函数类型不兼容: 期望 %v, 实际 %v", targetElem.Type(), originalFn.Type())
			c.LogErrorEvent(EventFunctionFailed, "函数绑定失败", map[string]any{
				"expected":      targetElem.Type().String(),
				"actual":        originalFn.Type().String(),
				"is_assignable": originalFn.Type().AssignableTo(targetElem.Type()),
				"function":      name,
				"error":         errMsg,
				"trace_id":      c.TraceID,
				"duration":      time.Since(startTime).Milliseconds(),
				"reason":        "type_mismatch",
				"action":        "bind",
			})
			return fmt.Errorf("%s", errMsg)
		}
		targetElem.Set(originalFn)
	}
	// 发布绑定成功事件
	duration := time.Since(startTime)
	c.PublishEventName(EventFunctionSuccess, map[string]any{
		"function":  name,
		"namespace": fn.Namespace,
		"is_method": fn.IsMethod,
		"action":    "bind",
		"trace_id":  c.TraceID,
		"duration":  duration.Milliseconds(),
		"caller":    c.AgentID,
	})
	// 更新统计信息
	c.mu.Lock()
	defer c.mu.Unlock()
	if fn.Stats != nil {
		fn.Stats.CallCount++
		fn.Stats.LastCalledAt = time.Now()
	}
	return nil
}

// GetWrapper 获取函数包装器
func (r *Context) GetWrapper(name string) (*FunctionWrapper, bool) {
	wrapper, ok := r.wrappers.Load(name)
	if !ok {
		return nil, false
	}
	return wrapper.(*FunctionWrapper), true
}

// ClearInterceptors 清除所有拦截器
func (r *Context) ClearInterceptors() {
	r.interceptor.Clear()
	r.wrappers.Range(func(key, value any) bool {
		value.(*FunctionWrapper).chain.Clear()
		return true
	})
}

// GlobalInterceptorCount 全局拦截器数量
func (r *Context) GlobalInterceptorCount() int {
	return r.interceptor.Count()
}

// FunctionInterceptorCount 函数拦截器数量
func (r *Context) FunctionInterceptorCount(name string) int {
	wrapper, ok := r.wrappers.Load(name)
	if !ok {
		return 0
	}
	return wrapper.(*FunctionWrapper).chain.Count()
}
