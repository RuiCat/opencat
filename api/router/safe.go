package router

import (
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// SafeGo 安全的协程执行
// fn: 要执行的函数
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SAFE-GO] goroutine panic recovered: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}

// WorkerPool 工作池限制协程数量
type WorkerPool struct {
	workers chan struct{}  // 工作信号量
	stop    chan struct{}  // 停止信号
	wg      sync.WaitGroup // 等待组
}

// NewWorkerPool 创建工作池
// maxWorkers: 最大工作协程数
// 返回: WorkerPool实例
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 100
	}
	return &WorkerPool{
		workers: make(chan struct{}, maxWorkers),
		stop:    make(chan struct{}),
	}
}

// Submit 提交任务到工作池
// task: 要执行的任务函数
func (wp *WorkerPool) Submit(task func()) {
	select {
	case wp.workers <- struct{}{}:
		wp.wg.Add(1)
		go wp.execute(task)
	case <-wp.stop:
		// 已关闭不接受新任务
	}
}

// execute 执行任务内部方法
// task: 要执行的任务
func (wp *WorkerPool) execute(task func()) {
	defer func() {
		<-wp.workers
		wp.wg.Done()
		if r := recover(); r != nil {
			log.Printf("[WORKER-POOL] panic recovered: %v\n%s", r, debug.Stack())
		}
	}()
	task()
}

// Shutdown 关闭工作池
func (wp *WorkerPool) Shutdown() {
	close(wp.stop)
	wp.wg.Wait()
}

// EventBatcher 事件批处理器
type EventBatcher struct {
	events    chan *Event    // 事件通道
	batchSize int            // 批次大小
	timeout   time.Duration  // 超时时间
	handler   func([]*Event) // 批次处理函数
	stop      chan struct{}  // 停止信号
	wg        sync.WaitGroup // 等待组
}

// NewEventBatcher 创建事件批处理器
func NewEventBatcher(batchSize int, timeout time.Duration, handler func([]*Event)) *EventBatcher {
	if batchSize <= 0 {
		batchSize = 10
	}
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}
	batcher := &EventBatcher{
		events:    make(chan *Event, batchSize*10),
		batchSize: batchSize,
		timeout:   timeout,
		handler:   handler,
		stop:      make(chan struct{}),
	}
	batcher.wg.Add(1)
	go batcher.process()
	return batcher
}

// Publish 发布事件
func (eb *EventBatcher) Publish(event *Event) {
	select {
	case eb.events <- event:
		// 事件入队
	default:
		log.Printf("[EVENT-BATCHER] queue full, dropping event: %s", event.Name)
	}
}

// process 处理批次
func (eb *EventBatcher) process() {
	defer eb.wg.Done()
	batch := make([]*Event, 0, eb.batchSize)
	timer := time.NewTimer(eb.timeout)
	defer timer.Stop()
	for {
		select {
		case event := <-eb.events:
			batch = append(batch, event)
			if len(batch) >= eb.batchSize {
				eb.flush(batch)
				batch = batch[:0]
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(eb.timeout)
			}
		case <-timer.C:
			if len(batch) > 0 {
				eb.flush(batch)
				batch = batch[:0]
			}
			timer.Reset(eb.timeout)
		case <-eb.stop:
			if len(batch) > 0 {
				eb.flush(batch)
			}
			return
		}
	}
}

// flush 刷新批次
func (eb *EventBatcher) flush(batch []*Event) {
	SafeGo(func() {
		eb.handler(batch)
	})
}

// Stop 停止批处理器
func (eb *EventBatcher) Stop() {
	close(eb.stop)
	eb.wg.Wait()
}

// 事件数据对象池
var eventDataPool = sync.Pool{
	New: func() any {
		return make(map[string]any, 8)
	},
}

// AcquireEventData 获取事件数据
func AcquireEventData() map[string]any {
	data := eventDataPool.Get().(map[string]any)
	data["timestamp"] = time.Now().UnixNano()
	return data
}

// ReleaseEventData 释放事件数据
func ReleaseEventData(data map[string]any) {
	for k := range data {
		delete(data, k)
	}
	eventDataPool.Put(data)
}

// SafePublish 安全发布事件
func (r *Router) SafePublish(eventName string, data map[string]any) {
	if r == nil || r.eventBus == nil {
		return
	}
	SafeGo(func() {
		event := &Event{
			Name:   eventName,
			Source: "router",
			Data:   data,
			Time:   time.Now(),
		}
		if traceID, ok := data["trace_id"].(string); ok {
			event.TraceID = traceID
		}
		r.eventBus.Publish(eventName, BlockTypeEvent, event)
		if r.config.EnableTriggers && r.triggerManager != nil {
			r.triggerManager.FireEvent(event)
		}
	})
}
