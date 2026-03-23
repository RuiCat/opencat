package router

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// AsyncExecutor 异步执行器接口
type AsyncExecutor interface {
	Execute(fn func() error) error                         // 执行无返回值的函数
	ExecuteWithResult(fn func() (any, error)) (any, error) // 执行有返回值的函数
	Shutdown()                                             // 关闭执行器
}

// SyncExecutor 同步执行器
type SyncExecutor struct{}

func (se *SyncExecutor) Execute(fn func() error) error {
	return fn()
}

func (se *SyncExecutor) ExecuteWithResult(fn func() (any, error)) (any, error) {
	return fn()
}

func (se *SyncExecutor) Shutdown() {}

// SafeAsyncExecutor 安全的异步执行器
type SafeAsyncExecutor struct {
	workerPool *WorkerPool
	stop       chan struct{}
	wg         sync.WaitGroup
}

// NewSafeAsyncExecutor 创建安全的异步执行器
func NewSafeAsyncExecutor(maxWorkers int) *SafeAsyncExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 100
	}
	return &SafeAsyncExecutor{
		workerPool: NewWorkerPool(maxWorkers),
		stop:       make(chan struct{}),
	}
}

// Execute 执行无返回值的函数
func (sae *SafeAsyncExecutor) Execute(fn func() error) error {
	sae.wg.Add(1)

	sae.workerPool.Submit(func() {
		defer sae.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ASYNC-EXECUTOR] panic recovered: %v\n%s", r, debug.Stack())
			}
		}()

		if err := fn(); err != nil {
			log.Printf("[ASYNC-EXECUTOR] function error: %v", err)
		}
	})

	return nil
}

// ExecuteWithResult 执行有返回值的函数
func (sae *SafeAsyncExecutor) ExecuteWithResult(fn func() (any, error)) (any, error) {
	resultChan := make(chan any, 1)
	errorChan := make(chan error, 1)

	sae.wg.Add(1)
	sae.workerPool.Submit(func() {
		defer sae.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ASYNC-EXECUTOR] panic recovered: %v\n%s", r, debug.Stack())
				errorChan <- &PanicError{Value: r}
			}
		}()

		result, err := fn()
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- result
	})

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, &TimeoutError{}
	}
}

// Shutdown 关闭执行器
func (sae *SafeAsyncExecutor) Shutdown() {
	close(sae.stop)
	sae.wg.Wait()
	sae.workerPool.Shutdown()
}

// PanicError panic错误
type PanicError struct {
	Value any
}

func (pe *PanicError) Error() string {
	return fmt.Sprintf("panic recovered: %v", pe.Value)
}

// TimeoutError 超时错误
type TimeoutError struct{}

func (te *TimeoutError) Error() string {
	return "async execution timeout"
}

// EventPublisher 事件发布器
type EventPublisher struct {
	executor    AsyncExecutor
	eventBus    *EventBus
	enableAsync bool
}

// NewEventPublisher 创建事件发布器
func NewEventPublisher(eventBus *EventBus, enableAsync bool) *EventPublisher {
	var executor AsyncExecutor
	if enableAsync {
		executor = NewSafeAsyncExecutor(50)
	} else {
		executor = &SyncExecutor{}
	}
	return &EventPublisher{
		executor:    executor,
		eventBus:    eventBus,
		enableAsync: enableAsync,
	}
}

// Publish 发布事件
func (ep *EventPublisher) Publish(event string, blockType BlockType, data any) {
	publishFn := func() error {
		ep.eventBus.publishInternal(event, blockType, data)
		return nil
	}
	ep.executor.Execute(publishFn)
}

// publishInternal 事件总线内部发布方法
func (eb *EventBus) publishInternal(event string, blockType BlockType, data any) {
	eb.mu.RLock()
	handlers := eb.subscribers[event]
	wildcardHandlers := eb.subscribers["*"]
	eb.mu.RUnlock()
	executor := NewSafeAsyncExecutor(20)
	defer executor.Shutdown()
	for _, handler := range handlers {
		executor.Execute(func() error {
			handler(blockType, data)
			return nil
		})
	}
	for _, handler := range wildcardHandlers {
		executor.Execute(func() error {
			handler(blockType, data)
			return nil
		})
	}
}

// RouterEventPublisher 路由器事件发布器
type RouterEventPublisher struct {
	router          *Router
	eventPublisher  *EventPublisher
	triggerExecutor AsyncExecutor
}

// NewRouterEventPublisher 创建路由器事件发布器
func NewRouterEventPublisher(router *Router, enableAsync bool) *RouterEventPublisher {
	return &RouterEventPublisher{
		router:          router,
		eventPublisher:  NewEventPublisher(router.eventBus, enableAsync),
		triggerExecutor: &SyncExecutor{},
	}
}

// PublishEvent 发布事件
func (rep *RouterEventPublisher) PublishEvent(event string, blockType BlockType, data any) {
	rep.eventPublisher.Publish(event, blockType, data)
}

// PublishEventName 发布事件名称
func (rep *RouterEventPublisher) PublishEventName(eventName string, data map[string]any) {
	if rep.router == nil {
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
	rep.eventPublisher.Publish(eventName, BlockTypeEvent, event)
	if rep.router.config.EnableTriggers && rep.router.triggerManager != nil {
		rep.triggerExecutor.Execute(func() error {
			return rep.router.triggerManager.FireEvent(event)
		})
	}
}

// Shutdown 关闭所有执行器
func (rep *RouterEventPublisher) Shutdown() {
	if sae, ok := rep.eventPublisher.executor.(*SafeAsyncExecutor); ok {
		sae.Shutdown()
	}
}
