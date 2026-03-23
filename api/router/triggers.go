package router

import (
	"fmt"
	"strings"
	"time"
)

// RegisterTrigger 注册触发器
func (tm *TriggerManager) RegisterTrigger(trigger *Trigger) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.triggers) >= tm.config.MaxTriggers {
		return fmt.Errorf("达到最大触发器数量限制: %d", tm.config.MaxTriggers)
	}
	if _, exists := tm.triggers[trigger.ID]; exists {
		return fmt.Errorf("触发器已存在: %s", trigger.ID)
	}
	if trigger.CreatedAt.IsZero() {
		trigger.CreatedAt = time.Now()
	}
	tm.triggers[trigger.ID] = trigger
	if tm.config.EnableStats {
		tm.stats.TotalTriggers = len(tm.triggers)
		if trigger.Enabled {
			tm.stats.EnabledTriggers++
		}
	}
	if tm.eventBus != nil {
		tm.eventBus.Publish(EventTriggerRegistered, BlockTypeEvent, map[string]any{
			"trigger_id":    trigger.ID,
			"trigger_name":  trigger.Name,
			"description":   trigger.Description,
			"event_pattern": trigger.EventPattern,
			"enabled":       trigger.Enabled,
			"priority":      trigger.Priority,
			"timestamp":     time.Now().UnixNano(),
		})
	}
	return nil
}

// UnregisterTrigger 注销触发器
func (tm *TriggerManager) UnregisterTrigger(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}
	if tm.config.EnableStats && trigger.Enabled {
		tm.stats.EnabledTriggers--
	}
	delete(tm.triggers, id)
	tm.stats.TotalTriggers = len(tm.triggers)
	if tm.eventBus != nil {
		tm.eventBus.Publish(EventTriggerUnregistered, BlockTypeEvent, map[string]any{
			"trigger_id":    id,
			"trigger_name":  trigger.Name,
			"description":   trigger.Description,
			"event_pattern": trigger.EventPattern,
			"enabled":       trigger.Enabled,
			"priority":      trigger.Priority,
			"fire_count":    trigger.FireCount,
			"success_count": trigger.SuccessCount,
			"error_count":   trigger.ErrorCount,
			"timestamp":     time.Now().UnixNano(),
		})
	}
	return nil
}

// EnableTrigger 启用触发器
func (tm *TriggerManager) EnableTrigger(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}
	if !trigger.Enabled {
		trigger.Enabled = true
		if tm.config.EnableStats {
			tm.stats.EnabledTriggers++
		}
		if tm.eventBus != nil {
			tm.eventBus.Publish(EventTriggerEnabled, BlockTypeEvent, map[string]any{
				"trigger_id":    id,
				"trigger_name":  trigger.Name,
				"description":   trigger.Description,
				"event_pattern": trigger.EventPattern,
				"priority":      trigger.Priority,
				"timestamp":     time.Now().UnixNano(),
			})
		}
	}
	return nil
}

// DisableTrigger 禁用触发器
func (tm *TriggerManager) DisableTrigger(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	trigger, exists := tm.triggers[id]
	if !exists {
		return fmt.Errorf("触发器不存在: %s", id)
	}
	if trigger.Enabled {
		trigger.Enabled = false
		if tm.config.EnableStats {
			tm.stats.EnabledTriggers--
		}
		if tm.eventBus != nil {
			tm.eventBus.Publish(EventTriggerDisabled, BlockTypeEvent, map[string]any{
				"trigger_id":    id,
				"trigger_name":  trigger.Name,
				"description":   trigger.Description,
				"event_pattern": trigger.EventPattern,
				"priority":      trigger.Priority,
				"timestamp":     time.Now().UnixNano(),
			})
		}
	}
	return nil
}

// GetTrigger 获取触发器
func (tm *TriggerManager) GetTrigger(id string) *Trigger {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.triggers[id]
}

// ListTriggers 列出所有触发器
func (tm *TriggerManager) ListTriggers() []*Trigger {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	triggers := make([]*Trigger, 0, len(tm.triggers))
	for _, trigger := range tm.triggers {
		triggers = append(triggers, trigger)
	}
	return triggers
}

// FireEvent 触发事件
func (tm *TriggerManager) FireEvent(event *Event) error {
	tm.mu.RLock()
	triggers := make([]*Trigger, 0)
	for _, trigger := range tm.triggers {
		if trigger.Enabled && tm.matchEventPattern(trigger.EventPattern, event.Name) {
			triggers = append(triggers, trigger)
		}
	}
	tm.mu.RUnlock()
	if tm.config.EnableStats {
		tm.mu.Lock()
		tm.stats.TotalEvents++
		tm.stats.LastEventTime = time.Now()
		tm.mu.Unlock()
	}
	for i := 0; i < len(triggers)-1; i++ {
		for j := 0; j < len(triggers)-i-1; j++ {
			if triggers[j].Priority < triggers[j+1].Priority {
				triggers[j], triggers[j+1] = triggers[j+1], triggers[j]
			}
		}
	}
	for _, trigger := range triggers {
		tm.executeTrigger(trigger, event)
	}
	return nil
}

// executeTrigger 执行单个触发器
func (tm *TriggerManager) executeTrigger(trigger *Trigger, event *Event) {
	if trigger.Condition != nil && !trigger.Condition(event) {
		return
	}
	tm.mu.Lock()
	trigger.FireCount++
	trigger.LastFired = time.Now()
	if tm.config.EnableStats {
		tm.stats.TriggeredCount++
	}
	tm.mu.Unlock()
	if tm.eventBus != nil {
		tm.eventBus.Publish(EventTriggerFired, BlockTypeEvent, map[string]any{
			"trigger_id":   trigger.ID,
			"trigger_name": trigger.Name,
			"event_name":   event.Name,
			"event_source": event.Source,
			"fire_count":   trigger.FireCount,
			"timestamp":    time.Now().UnixNano(),
		})
	}
	if trigger.Action != nil {
		err := trigger.Action(event)
		tm.mu.Lock()
		if err != nil {
			trigger.ErrorCount++
			trigger.LastError = err.Error()
			if tm.config.EnableStats {
				tm.stats.ErrorCount++
			}
			if tm.eventBus != nil {
				tm.eventBus.Publish(EventTriggerError, BlockTypeEvent, map[string]any{
					"trigger_id":   trigger.ID,
					"trigger_name": trigger.Name,
					"event_name":   event.Name,
					"error":        err.Error(),
					"timestamp":    time.Now().UnixNano(),
				})
			}
		} else {
			trigger.SuccessCount++
			if tm.config.EnableStats {
				tm.stats.SuccessCount++
			}
		}
		tm.mu.Unlock()
	}
}

// matchEventPattern 匹配事件模式
func (tm *TriggerManager) matchEventPattern(pattern, eventName string) bool {
	if pattern == eventName {
		return true
	}
	if pattern == "*" {
		return true
	}
	if before, ok := strings.CutSuffix(pattern, ".*"); ok {
		prefix := before
		return strings.HasPrefix(eventName, prefix+".")
	}
	if after, ok := strings.CutPrefix(pattern, "*."); ok {
		suffix := after
		return strings.HasSuffix(eventName, "."+suffix)
	}
	return false
}

// GetStats 获取统计信息
func (tm *TriggerManager) GetStats() TriggerManagerStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return *tm.stats
}
