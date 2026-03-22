package router

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(event string, handler EventHandler) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if eb.subscribers[event] == nil {
		eb.subscribers[event] = make([]EventHandler, 0)
	}
	eb.subscribers[event] = append(eb.subscribers[event], handler)
	return len(eb.subscribers[event]) - 1
}

// Publish 发布事件
func (eb *EventBus) Publish(event string, blockType BlockType, data any) {
	eb.mu.RLock()
	handlers := eb.subscribers[event]
	wildcardHandlers := eb.subscribers["*"]
	eb.mu.RUnlock()
	for _, handler := range handlers {
		go handler(blockType, data)
	}
	for _, handler := range wildcardHandlers {
		go handler(blockType, data)
	}
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	handlers := eb.subscribers[event]
	for _, handler := range handlers {
		go handler(BlockTypeUnsubscribe, nil)
	}
	delete(eb.subscribers, event)
}

// HasSubscribers 检查事件是否有订阅者
func (eb *EventBus) HasSubscribers(event string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	handlers, exists := eb.subscribers[event]
	return exists && len(handlers) > 0
}

// SubscriberCount 获取订阅者数量
func (eb *EventBus) SubscriberCount(event string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers[event])
}
