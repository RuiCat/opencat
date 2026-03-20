package router

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// 触发器管理器
// ============================================================================

// TriggerManager 触发器管理器
type TriggerManager struct {
	mu       sync.RWMutex
	triggers map[string]*Trigger // 触发器注册表，key为触发器ID

	// 事件总线引用
	eventBus *EventBus

	// 统计
	stats *TriggerManagerStats

	// 配置
	config *TriggerManagerConfig
}

// TriggerManagerStats 触发器管理器统计
type TriggerManagerStats struct {
	TotalTriggers   int       `json:"total_triggers"`
	EnabledTriggers int       `json:"enabled_triggers"`
	TotalEvents     int64     `json:"total_events"`
	TriggeredCount  int64     `json:"triggered_count"`
	SuccessCount    int64     `json:"success_count"`
	ErrorCount      int64     `json:"error_count"`
	StartTime       time.Time `json:"start_time"`
	LastEventTime   time.Time `json:"last_event_time"`
}

// TriggerManagerConfig 触发器管理器配置
type TriggerManagerConfig struct {
	MaxTriggers        int  `json:"max_triggers"`         // 最大触发器数量
	EnableAsync        bool `json:"enable_async"`         // 是否启用异步执行
	MaxConcurrentFires int  `json:"max_concurrent_fires"` // 最大并发触发数
	EventBufferSize    int  `json:"event_buffer_size"`    // 事件缓冲区大小
	EnableStats        bool `json:"enable_stats"`         // 是否启用统计
}

// DefaultTriggerManagerConfig 默认配置
func DefaultTriggerManagerConfig() *TriggerManagerConfig {
	return &TriggerManagerConfig{
		MaxTriggers:        1000,
		EnableAsync:        true,
		MaxConcurrentFires: 100,
		EventBufferSize:    1000,
		EnableStats:        true,
	}
}

// NewTriggerManager 创建新的触发器管理器
func NewTriggerManager(eventBus *EventBus, config *TriggerManagerConfig) *TriggerManager {
	if config == nil {
		config = DefaultTriggerManagerConfig()
	}

	return &TriggerManager{
		triggers: make(map[string]*Trigger),
		eventBus: eventBus,
		stats: &TriggerManagerStats{
			StartTime: time.Now(),
		},
		config: config,
	}
}

// ============================================================================
// 触发器管理
// ============================================================================

// Register 注册触发器
func (tm *TriggerManager) Register(trigger *Trigger) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查触发器数量限制
	if len(tm.triggers) >= tm.config.MaxTriggers {
		return fmt.Errorf("达到最大触发器数量限制: %d", tm.config.MaxTriggers)
	}

	// 检查ID是否已存在
	if _, exists := tm.triggers[trigger.ID]; exists {
		return fmt.Errorf("触发器ID已存在: %s", trigger.ID)
	}

	// 注册触发器
	tm.triggers[trigger.ID] = trigger

	// 更新统计
	tm.stats.TotalTriggers++
	if trigger.Enabled {
		tm.stats.EnabledTriggers++
	}

	// 订阅事件总线
	if tm.eventBus != nil {
		// 使用通配符订阅所有事件，在HandleEvent中进行模式匹配
		tm.eventBus.Subscribe("*", func(eventName string, data interface{}) {
			// 将旧格式事件转换为新格式
			if event, ok := data.(*Event); ok {
				tm.HandleEvent(event)
			}
		})
	}

	return nil
}

// Unregister 注销触发器
func (tm *TriggerManager) Unregister(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}

	// 更新统计
	tm.stats.TotalTriggers--
	if trigger.Enabled {
		tm.stats.EnabledTriggers--
	}

	// 删除触发器
	delete(tm.triggers, id)

	return nil
}

// Enable 启用触发器
func (tm *TriggerManager) Enable(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}

	if !trigger.Enabled {
		trigger.Enabled = true
		tm.stats.EnabledTriggers++
	}

	return nil
}

// Disable 禁用触发器
func (tm *TriggerManager) Disable(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}

	if trigger.Enabled {
		trigger.Enabled = false
		tm.stats.EnabledTriggers--
	}

	return nil
}

// Get 获取触发器
func (tm *TriggerManager) Get(id string) (*Trigger, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	trigger, exists := tm.triggers[id]
	if !exists {
		return nil, fmt.Errorf("触发器不存在: %s", id)
	}

	return trigger, nil
}

// List 列出所有触发器
func (tm *TriggerManager) List() []*Trigger {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	triggers := make([]*Trigger, 0, len(tm.triggers))
	for _, trigger := range tm.triggers {
		triggers = append(triggers, trigger)
	}

	// 按优先级排序
	sort.Slice(triggers, func(i, j int) bool {
		return triggers[i].Priority > triggers[j].Priority
	})

	return triggers
}

// ListByPattern 按事件模式列出触发器
func (tm *TriggerManager) ListByPattern(pattern string) []*Trigger {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	triggers := make([]*Trigger, 0)
	for _, trigger := range tm.triggers {
		if matchPattern(pattern, trigger.EventPattern) {
			triggers = append(triggers, trigger)
		}
	}

	// 按优先级排序
	sort.Slice(triggers, func(i, j int) bool {
		return triggers[i].Priority > triggers[j].Priority
	})

	return triggers
}

// ============================================================================
// 事件处理
// ============================================================================

// HandleEvent 处理事件
func (tm *TriggerManager) HandleEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	// 更新统计
	tm.stats.TotalEvents++
	tm.stats.LastEventTime = time.Now()

	// 获取匹配的触发器
	triggers := tm.getMatchingTriggers(event)
	if len(triggers) == 0 {
		return nil
	}

	// 执行触发器
	var lastError error
	for _, trigger := range triggers {
		if err := tm.fireTrigger(trigger, event); err != nil {
			lastError = err

			// 发布触发器错误事件
			tm.publishTriggerEvent(EventTriggerError, trigger, event, err)
		} else {
			// 发布触发器触发事件
			tm.publishTriggerEvent(EventTriggerFired, trigger, event, nil)
		}
	}

	return lastError
}

// getMatchingTriggers 获取匹配事件的触发器
func (tm *TriggerManager) getMatchingTriggers(event *Event) []*Trigger {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	matching := make([]*Trigger, 0)

	for _, trigger := range tm.triggers {
		if trigger.Check(event) {
			matching = append(matching, trigger)
		}
	}

	// 按优先级排序（高的先执行）
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].Priority > matching[j].Priority
	})

	return matching
}

// fireTrigger 触发单个触发器
func (tm *TriggerManager) fireTrigger(trigger *Trigger, event *Event) error {
	if tm.config.EnableAsync {
		// 异步执行
		go func() {
			if err := trigger.Fire(event); err != nil {
				// 记录错误但不阻塞
				fmt.Printf("[触发器错误] %s: %v\n", trigger.Name, err)
			}
		}()
		return nil
	}

	// 同步执行
	return trigger.Fire(event)
}

// publishTriggerEvent 发布触发器相关事件
func (tm *TriggerManager) publishTriggerEvent(eventName string, trigger *Trigger, originalEvent *Event, err error) {
	if tm.eventBus == nil {
		return
	}

	data := map[string]interface{}{
		"trigger_id":   trigger.ID,
		"trigger_name": trigger.Name,
		"event":        originalEvent,
		"time":         time.Now(),
	}

	if err != nil {
		data["error"] = err.Error()
	}

	// 发布事件
	tm.eventBus.Publish(eventName, NewEvent(eventName, "trigger_manager", data))
}

// ============================================================================
// 统计和状态
// ============================================================================

// GetStats 获取统计信息
func (tm *TriggerManager) GetStats() *TriggerManagerStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := *tm.stats
	return &stats
}

// GetTriggerStats 获取触发器统计
func (tm *TriggerManager) GetTriggerStats(id string) (*Trigger, error) {
	trigger, err := tm.Get(id)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

// ResetStats 重置统计
func (tm *TriggerManager) ResetStats() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.stats = &TriggerManagerStats{
		StartTime: time.Now(),
	}

	// 重置所有触发器的统计
	for _, trigger := range tm.triggers {
		trigger.FireCount = 0
		trigger.SuccessCount = 0
		trigger.ErrorCount = 0
		trigger.LastError = ""
	}
}

// ============================================================================
// 工具方法
// ============================================================================

// PublishEvent 发布事件（便捷方法）
func (tm *TriggerManager) PublishEvent(event *Event) error {
	if tm.eventBus == nil {
		return fmt.Errorf("事件总线未设置")
	}

	// 发布到事件总线
	tm.eventBus.Publish(event.Name, event)

	// 同时处理事件
	return tm.HandleEvent(event)
}

// CreateAndRegister 创建并注册触发器（便捷方法）
func (tm *TriggerManager) CreateAndRegister(id, name, description, eventPattern string) (*Trigger, error) {
	trigger := NewTrigger(id, name, description, eventPattern)
	return trigger, tm.Register(trigger)
}

// CreateRuleTrigger 创建规则触发器（便捷方法）
func (tm *TriggerManager) CreateRuleTrigger(ruleID, ruleName, eventPattern string, condition func(*Event) bool, action func(*Event) error) (*Trigger, error) {
	trigger := NewTrigger(ruleID, ruleName, "规则触发器: "+ruleName, eventPattern).
		WithCondition(condition).
		WithAction(action)

	return trigger, tm.Register(trigger)
}

// Clear 清空所有触发器
func (tm *TriggerManager) Clear() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.triggers = make(map[string]*Trigger)
	tm.stats.TotalTriggers = 0
	tm.stats.EnabledTriggers = 0
}
