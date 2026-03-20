package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// 事件类型定义
// ============================================================================

// EventType 事件类型枚举
type EventType string

const (
	// 插件生命周期事件
	EventPluginLoaded   EventType = "plugin.loaded"
	EventPluginUnloaded EventType = "plugin.unloaded"
	EventPluginEnabled  EventType = "plugin.enabled"
	EventPluginDisabled EventType = "plugin.disabled"

	// 插件错误事件
	EventPluginLoadError     EventType = "plugin.load.error"
	EventPluginUnloadError   EventType = "plugin.unload.error"
	EventPluginInitError     EventType = "plugin.init.error"
	EventPluginShutdownError EventType = "plugin.shutdown.error"

	// 工具事件
	EventToolRegistered   EventType = "tool.registered"
	EventToolUnregistered EventType = "tool.unregistered"
	EventToolCalled       EventType = "tool.called"
	EventToolCallError    EventType = "tool.call.error"

	// 依赖事件
	EventDependencyMissing   EventType = "dependency.missing"
	EventCircularDependency  EventType = "dependency.circular"
	EventDependencySatisfied EventType = "dependency.satisfied"
	EventDependencyError     EventType = "dependency.error"

	// 路由事件
	EventFunctionRegistered   EventType = "function.registered"
	EventFunctionUnregistered EventType = "function.unregistered"
	EventFunctionCalled       EventType = "function.called"
	EventFunctionSuccess      EventType = "function.success"
	EventFunctionFailed       EventType = "function.failed"

	// 系统事件
	EventSystemStartup   EventType = "system.startup"
	EventSystemShutdown  EventType = "system.shutdown"
	EventSystemError     EventType = "system.error"
	EventSystemRecovered EventType = "system.recovered"
)

// ============================================================================
// 错误事件数据结构
// ============================================================================

// ErrorEventData 错误事件数据
type ErrorEventData struct {
	PluginName   string                 `json:"plugin_name,omitempty"`
	ToolName     string                 `json:"tool_name,omitempty"`
	FunctionName string                 `json:"function_name,omitempty"`
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	Stack        string                 `json:"stack,omitempty"`
	Recoverable  bool                   `json:"recoverable"`
	Retryable    bool                   `json:"retryable"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Severity     string                 `json:"severity"` // info, warning, error, critical
}

// NewErrorEventData 创建错误事件数据
func NewErrorEventData(errorCode, errorMessage string) *ErrorEventData {
	return &ErrorEventData{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		Recoverable:  true,
		Retryable:    false,
		Timestamp:    time.Now(),
		Severity:     "error",
		Context:      make(map[string]interface{}),
	}
}

// WithPlugin 设置插件信息
func (d *ErrorEventData) WithPlugin(name string) *ErrorEventData {
	d.PluginName = name
	return d
}

// WithTool 设置工具信息
func (d *ErrorEventData) WithTool(name string) *ErrorEventData {
	d.ToolName = name
	return d
}

// WithFunction 设置函数信息
func (d *ErrorEventData) WithFunction(name string) *ErrorEventData {
	d.FunctionName = name
	return d
}

// WithStack 设置堆栈信息
func (d *ErrorEventData) WithStack(stack string) *ErrorEventData {
	d.Stack = stack
	return d
}

// WithContext 添加上下文信息
func (d *ErrorEventData) WithContext(key string, value interface{}) *ErrorEventData {
	if d.Context == nil {
		d.Context = make(map[string]interface{})
	}
	d.Context[key] = value
	return d
}

// WithSeverity 设置严重程度
func (d *ErrorEventData) WithSeverity(severity string) *ErrorEventData {
	d.Severity = severity
	return d
}

// MarkUnrecoverable 标记为不可恢复
func (d *ErrorEventData) MarkUnrecoverable() *ErrorEventData {
	d.Recoverable = false
	return d
}

// MarkRetryable 标记为可重试
func (d *ErrorEventData) MarkRetryable() *ErrorEventData {
	d.Retryable = true
	return d
}

// ============================================================================
// 事件工厂函数
// ============================================================================

// NewPluginErrorEvent 创建插件错误事件
func NewPluginErrorEvent(eventType EventType, pluginName string, err error) *Event {
	errorData := NewErrorEventData("PLUGIN_ERROR", err.Error()).
		WithPlugin(pluginName).
		WithSeverity("error")

	return &Event{
		Type:      string(eventType),
		Name:      fmt.Sprintf("%s.%s", pluginName, eventType),
		Data:      errorData,
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	}
}

// NewToolErrorEvent 创建工具错误事件
func NewToolErrorEvent(pluginName, toolName string, err error) *Event {
	errorData := NewErrorEventData("TOOL_ERROR", err.Error()).
		WithPlugin(pluginName).
		WithTool(toolName).
		WithSeverity("warning")

	return &Event{
		Type:      string(EventToolCallError),
		Name:      fmt.Sprintf("%s.%s.error", pluginName, toolName),
		Data:      errorData,
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	}
}

// NewDependencyErrorEvent 创建依赖错误事件
func NewDependencyErrorEvent(pluginName string, dependency string, err error) *Event {
	errorData := NewErrorEventData("DEPENDENCY_ERROR", err.Error()).
		WithPlugin(pluginName).
		WithContext("dependency", dependency).
		WithSeverity("error")

	return &Event{
		Type:      string(EventDependencyError),
		Name:      fmt.Sprintf("%s.dependency.error", pluginName),
		Data:      errorData,
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	}
}

// NewSystemErrorEvent 创建系统错误事件
func NewSystemErrorEvent(err error) *Event {
	errorData := NewErrorEventData("SYSTEM_ERROR", err.Error()).
		WithSeverity("critical").
		MarkUnrecoverable()

	return &Event{
		Type:      string(EventSystemError),
		Name:      "system.error",
		Data:      errorData,
		Timestamp: time.Now(),
		Source:    "system",
	}
}

// ============================================================================
// 事件处理器增强
// ============================================================================

// ErrorEventHandler 错误事件处理器
type ErrorEventHandler func(ctx context.Context, event *Event) error

// EventManager 事件管理器
type EventManager struct {
	mu            sync.RWMutex
	handlers      map[EventType][]EventHandler
	errorHandlers map[EventType][]ErrorEventHandler
	maskConfig    *EventMaskConfig
	router        Router
}

// EventMaskConfig 事件掩盖配置
type EventMaskConfig struct {
	Enabled          bool
	MaskErrorDetails bool
	MaskStackTraces  bool
	MaskPluginNames  bool
	MaskToolNames    bool
	AllowedEvents    map[EventType]bool
}

// NewEventManager 创建事件管理器
func NewEventManager(router Router) *EventManager {
	return &EventManager{
		handlers:      make(map[EventType][]EventHandler),
		errorHandlers: make(map[EventType][]ErrorEventHandler),
		maskConfig: &EventMaskConfig{
			Enabled:          true,
			MaskErrorDetails: true,
			MaskStackTraces:  true,
			MaskPluginNames:  false,
			MaskToolNames:    false,
			AllowedEvents: map[EventType]bool{
				EventPluginLoaded:     true,
				EventPluginUnloaded:   true,
				EventPluginEnabled:    true,
				EventPluginDisabled:   true,
				EventToolRegistered:   true,
				EventToolUnregistered: true,
				EventToolCalled:       true,
				EventFunctionSuccess:  true,
			},
		},
		router: router,
	}
}

// Emit 发送事件
func (em *EventManager) Emit(ctx context.Context, event *Event) error {
	eventType := EventType(event.Type)

	// 检查是否需要掩盖
	if em.maskConfig.Enabled && em.shouldMaskEvent(eventType) {
		event = em.maskEvent(event)
	}

	// 发送到路由
	if em.router != nil {
		if err := em.router.PublishEvent(event.Type, event.Data); err != nil {
			return err
		}
	}

	// 调用普通事件处理器
	em.mu.RLock()
	handlers := em.handlers[eventType]
	em.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(event); err != nil {
				// 记录处理器错误但不中断流程
				fmt.Printf("事件处理器错误: %v\n", err)
			}
		}(handler)
	}

	// 调用错误事件处理器（如果是错误事件）
	if em.isErrorEvent(eventType) {
		em.mu.RLock()
		errorHandlers := em.errorHandlers[eventType]
		em.mu.RUnlock()

		for _, handler := range errorHandlers {
			go func(h ErrorEventHandler) {
				if err := h(ctx, event); err != nil {
					// 记录错误处理器错误
					fmt.Printf("错误事件处理器错误: %v\n", err)
				}
			}(handler)
		}
	}

	return nil
}

// Subscribe 订阅事件
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.handlers[eventType] = append(em.handlers[eventType], handler)
	return nil
}

// SubscribeError 订阅错误事件
func (em *EventManager) SubscribeError(eventType EventType, handler ErrorEventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.errorHandlers[eventType] = append(em.errorHandlers[eventType], handler)
	return nil
}

// Unsubscribe 取消订阅事件
func (em *EventManager) Unsubscribe(eventType EventType, handler EventHandler) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	handlers := em.handlers[eventType]
	for i, h := range handlers {
		if &h == &handler {
			em.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// shouldMaskEvent 判断是否需要掩盖事件
func (em *EventManager) shouldMaskEvent(eventType EventType) bool {
	// 如果事件在允许列表中，不需要掩盖
	if em.maskConfig.AllowedEvents[eventType] {
		return false
	}

	// 错误事件需要掩盖
	return em.isErrorEvent(eventType)
}

// isErrorEvent 判断是否是错误事件
func (em *EventManager) isErrorEvent(eventType EventType) bool {
	errorEvents := map[EventType]bool{
		EventPluginLoadError:     true,
		EventPluginUnloadError:   true,
		EventPluginInitError:     true,
		EventPluginShutdownError: true,
		EventToolCallError:       true,
		EventDependencyMissing:   true,
		EventCircularDependency:  true,
		EventDependencyError:     true,
		EventFunctionFailed:      true,
		EventSystemError:         true,
	}

	return errorEvents[eventType]
}

// maskEvent 掩盖事件
func (em *EventManager) maskEvent(event *Event) *Event {
	maskedEvent := &Event{
		Type:      "internal." + event.Type,
		Name:      event.Name,
		Timestamp: event.Timestamp,
		Source:    "masked." + event.Source,
	}

	// 掩盖事件数据
	if data, ok := event.Data.(*ErrorEventData); ok {
		maskedData := &ErrorEventData{
			ErrorCode:    data.ErrorCode,
			ErrorMessage: "发生内部错误，已记录到日志",
			Timestamp:    data.Timestamp,
			Severity:     data.Severity,
			Recoverable:  data.Recoverable,
			Retryable:    data.Retryable,
		}

		// 根据配置掩盖详细信息
		if !em.maskConfig.MaskPluginNames {
			maskedData.PluginName = data.PluginName
		}
		if !em.maskConfig.MaskToolNames {
			maskedData.ToolName = data.ToolName
		}
		if !em.maskConfig.MaskErrorDetails {
			maskedData.ErrorMessage = data.ErrorMessage
		}
		if !em.maskConfig.MaskStackTraces {
			maskedData.Stack = data.Stack
		}

		maskedEvent.Data = maskedData
	} else {
		// 非错误事件数据掩盖
		maskedEvent.Data = map[string]interface{}{
			"type":        "masked_event",
			"description": "内部事件，详细信息已记录",
			"timestamp":   time.Now().Unix(),
		}
	}

	return maskedEvent
}

// SetMaskConfig 设置掩盖配置
func (em *EventManager) SetMaskConfig(config *EventMaskConfig) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.maskConfig = config
}

// GetMaskConfig 获取掩盖配置
func (em *EventManager) GetMaskConfig() *EventMaskConfig {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.maskConfig
}

// ============================================================================
// 全局事件管理器
// ============================================================================

var (
	globalEventManager *EventManager
	eventManagerOnce   sync.Once
)

// GetGlobalEventManager 获取全局事件管理器
func GetGlobalEventManager(router Router) *EventManager {
	eventManagerOnce.Do(func() {
		globalEventManager = NewEventManager(router)
	})
	return globalEventManager
}
