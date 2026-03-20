package router

import (
	"time"
)

// ============================================================================
// 事件系统
// ============================================================================

// Event 事件定义
type Event struct {
	Name     string                 `json:"name"`     // 事件名称
	Source   string                 `json:"source"`   // 事件来源
	Data     map[string]interface{} `json:"data"`     // 事件数据
	Time     time.Time              `json:"time"`     // 发生时间
	TraceID  string                 `json:"trace_id"` // 追踪ID
	Metadata map[string]string      `json:"metadata"` // 元数据
}

// NewEvent 创建新事件
func NewEvent(name, source string, data map[string]interface{}) *Event {
	return &Event{
		Name:     name,
		Source:   source,
		Data:     data,
		Time:     time.Now(),
		TraceID:  generateTraceID(),
		Metadata: make(map[string]string),
	}
}

// WithMetadata 添加元数据
func (e *Event) WithMetadata(key, value string) *Event {
	e.Metadata[key] = value
	return e
}

// WithTraceID 设置追踪ID
func (e *Event) WithTraceID(traceID string) *Event {
	e.TraceID = traceID
	return e
}

// ============================================================================
// 触发器系统
// ============================================================================

// Trigger 触发器定义
type Trigger struct {
	ID           string `json:"id"`            // 触发器ID
	Name         string `json:"name"`          // 触发器名称
	Description  string `json:"description"`   // 描述
	EventPattern string `json:"event_pattern"` // 事件模式（支持通配符）

	// 触发条件
	Condition func(event *Event) bool `json:"-"` // 条件检查函数

	// 执行动作
	Action func(event *Event) error `json:"-"` // 动作执行函数

	// 配置
	Enabled   bool      `json:"enabled"`    // 是否启用
	Priority  int       `json:"priority"`   // 优先级（高的先执行）
	CreatedAt time.Time `json:"created_at"` // 创建时间
	LastFired time.Time `json:"last_fired"` // 上次触发时间
	FireCount int       `json:"fire_count"` // 触发次数

	// 统计
	SuccessCount int    `json:"success_count"` // 成功执行次数
	ErrorCount   int    `json:"error_count"`   // 失败执行次数
	LastError    string `json:"last_error"`    // 最后错误信息
}

// NewTrigger 创建新触发器
func NewTrigger(id, name, description, eventPattern string) *Trigger {
	return &Trigger{
		ID:           id,
		Name:         name,
		Description:  description,
		EventPattern: eventPattern,
		Enabled:      true,
		Priority:     50, // 默认优先级
		CreatedAt:    time.Now(),
	}
}

// WithCondition 设置条件函数
func (t *Trigger) WithCondition(condition func(event *Event) bool) *Trigger {
	t.Condition = condition
	return t
}

// WithAction 设置动作函数
func (t *Trigger) WithAction(action func(event *Event) error) *Trigger {
	t.Action = action
	return t
}

// WithPriority 设置优先级
func (t *Trigger) WithPriority(priority int) *Trigger {
	t.Priority = priority
	return t
}

// WithEnabled 设置启用状态
func (t *Trigger) WithEnabled(enabled bool) *Trigger {
	t.Enabled = enabled
	return t
}

// Check 检查触发器是否应该触发
func (t *Trigger) Check(event *Event) bool {
	if !t.Enabled {
		return false
	}

	// 检查事件模式匹配
	if !matchPattern(t.EventPattern, event.Name) {
		return false
	}

	// 如果有条件函数，检查条件
	if t.Condition != nil {
		return t.Condition(event)
	}

	// 没有条件函数时，总是触发
	return true
}

// Fire 触发触发器
func (t *Trigger) Fire(event *Event) error {
	if !t.Enabled {
		return nil
	}

	t.LastFired = time.Now()
	t.FireCount++

	if t.Action == nil {
		return nil // 没有动作，直接返回
	}

	// 执行动作
	err := t.Action(event)
	if err != nil {
		t.ErrorCount++
		t.LastError = err.Error()
		return err
	}

	t.SuccessCount++
	return nil
}

// ============================================================================
// 内置事件类型
// ============================================================================

const (
	// 路由事件
	EventRouterStarted = "router.started" // 路由启动
	EventRouterStopped = "router.stopped" // 路由停止

	// 函数调用事件
	EventFunctionCalled  = "function.called"  // 函数被调用
	EventFunctionSuccess = "function.success" // 函数调用成功
	EventFunctionFailed  = "function.failed"  // 函数调用失败
	EventFunctionPanic   = "function.panic"   // 函数发生panic

	// 定时事件
	EventTimerTick = "timer.tick" // 定时器触发

	// 触发器事件
	EventTriggerFired = "trigger.fired" // 触发器触发
	EventTriggerError = "trigger.error" // 触发器错误

	// 系统事件
	EventSystemWarning = "system.warning" // 系统警告
	EventSystemError   = "system.error"   // 系统错误
	EventSystemInfo    = "system.info"    // 系统信息
)

// ============================================================================
// 辅助函数
// ============================================================================

// matchPattern 检查事件名称是否匹配模式
// 支持通配符：* 匹配任意字符
func matchPattern(pattern, eventName string) bool {
	if pattern == "*" {
		return true
	}

	// 简单通配符匹配
	if pattern == eventName {
		return true
	}

	// 处理通配符
	patternRunes := []rune(pattern)
	eventRunes := []rune(eventName)

	patternIdx := 0
	eventIdx := 0
	starIdx := -1
	matchIdx := -1

	for eventIdx < len(eventRunes) {
		if patternIdx < len(patternRunes) && (patternRunes[patternIdx] == eventRunes[eventIdx] || patternRunes[patternIdx] == '*') {
			if patternRunes[patternIdx] == '*' {
				starIdx = patternIdx
				matchIdx = eventIdx
				patternIdx++
			} else {
				patternIdx++
				eventIdx++
			}
		} else if starIdx != -1 {
			patternIdx = starIdx + 1
			matchIdx++
			eventIdx = matchIdx
		} else {
			return false
		}
	}

	// 跳过末尾的*
	for patternIdx < len(patternRunes) && patternRunes[patternIdx] == '*' {
		patternIdx++
	}

	return patternIdx == len(patternRunes)
}

// IsBuiltinEvent 检查是否是内置事件
func IsBuiltinEvent(eventName string) bool {
	switch eventName {
	case EventRouterStarted, EventRouterStopped,
		EventFunctionCalled, EventFunctionSuccess, EventFunctionFailed, EventFunctionPanic,
		EventTimerTick,
		EventTriggerFired, EventTriggerError,
		EventSystemWarning, EventSystemError, EventSystemInfo:
		return true
	default:
		return false
	}
}
