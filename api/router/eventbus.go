package router

import (
	"sync"
)

// EventBus 事件总线
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if _, exists := eb.subscribers[event]; !exists {
		eb.subscribers[event] = make([]EventHandler, 0)
	}

	eb.subscribers[event] = append(eb.subscribers[event], handler)
}

// Unsubscribe 取消订阅事件
func (eb *EventBus) Unsubscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers, exists := eb.subscribers[event]
	if !exists {
		return
	}

	for i, h := range handlers {
		// 比较函数指针（注意：这种方法有限制）
		// 在实际应用中可能需要更复杂的比较逻辑
		if &h == &handler {
			eb.subscribers[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// 如果没有订阅者了，删除该事件
	if len(eb.subscribers[event]) == 0 {
		delete(eb.subscribers, event)
	}
}

// Publish 发布事件
func (eb *EventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	handlers, exists := eb.subscribers[event]
	eb.mu.RUnlock()

	if !exists {
		return
	}

	// 异步执行所有处理器
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// 记录panic但不影响其他处理器
					// 在实际应用中应该记录日志
				}
			}()
			h(event, data)
		}(handler)
	}
}

// GetSubscriberCount 获取订阅者数量
func (eb *EventBus) GetSubscriberCount(event string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, exists := eb.subscribers[event]
	if !exists {
		return 0
	}

	return len(handlers)
}

// ListEvents 列出所有事件
func (eb *EventBus) ListEvents() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	events := make([]string, 0, len(eb.subscribers))
	for event := range eb.subscribers {
		events = append(events, event)
	}

	return events
}

// Clear 清空所有订阅
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers = make(map[string][]EventHandler)
}

// ClearEvent 清空特定事件的订阅
func (eb *EventBus) ClearEvent(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.subscribers, event)
}
