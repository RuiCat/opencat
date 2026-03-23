package router

import (
	"fmt"
	"sync"
	"time"
)

// EventDataBuilder 事件数据构建器
type EventDataBuilder struct {
	data map[string]any // 事件数据
}

// NewEventData 创建事件数据
// 返回: 事件数据构建器实例
func NewEventData() *EventDataBuilder {
	return &EventDataBuilder{
		data: map[string]any{
			"timestamp": time.Now().UnixNano(),
		},
	}
}

// With 添加数据字段
func (b *EventDataBuilder) With(key string, value any) *EventDataBuilder {
	b.data[key] = value
	return b
}

// WithTraceID 添加追踪ID
func (b *EventDataBuilder) WithTraceID(traceID string) *EventDataBuilder {
	if traceID != "" {
		b.data["trace_id"] = traceID
	}
	return b
}

// WithFunctionInfo 添加函数信息
func (b *EventDataBuilder) WithFunctionInfo(fn *Function) *EventDataBuilder {
	b.data["name"] = fn.Name
	b.data["namespace"] = fn.Namespace
	b.data["description"] = fn.Description
	b.data["builtin"] = fn.Builtin
	b.data["is_method"] = fn.IsMethod
	return b
}

// Build 构建最终数据
func (b *EventDataBuilder) Build() map[string]any {
	return b.data
}

// CheckExists 检查是否存在
func CheckExists[T any](m map[string]T, key, entity string) error {
	if _, exists := m[key]; exists {
		return fmt.Errorf("%s已存在: %s", entity, key)
	}
	return nil
}

// CheckNotExists 检查是否不存在
func CheckNotExists[T any](m map[string]T, key, entity string) error {
	if _, exists := m[key]; !exists {
		return fmt.Errorf("%s不存在: %s", entity, key)
	}
	return nil
}

// WithLock 通用锁操作
func WithLock(mu *sync.RWMutex, write bool, fn func()) {
	if write {
		mu.Lock()
		defer mu.Unlock()
	} else {
		mu.RLock()
		defer mu.RUnlock()
	}
	fn()
}

// WithWriteLock 写锁包装
func WithWriteLock(mu *sync.RWMutex, fn func()) {
	WithLock(mu, true, fn)
}

// WithReadLock 读锁包装
func WithReadLock(mu *sync.RWMutex, fn func()) {
	WithLock(mu, false, fn)
}

// ErrorBuilder 错误构建器
type ErrorBuilder struct {
	code    string
	message string
}

// NewError 创建错误
func NewError(code string) *ErrorBuilder {
	return &ErrorBuilder{code: code}
}

// WithMessage 设置错误消息
func (b *ErrorBuilder) WithMessage(message string) *ErrorBuilder {
	b.message = message
	return b
}

// WithFormat 格式化错误消息
func (b *ErrorBuilder) WithFormat(format string, args ...any) *ErrorBuilder {
	b.message = fmt.Sprintf(format, args...)
	return b
}

// Build 构建错误
func (b *ErrorBuilder) Build() error {
	if b.message == "" {
		b.message = b.code
	}
	return fmt.Errorf("%s: %s", b.code, b.message)
}

// Timer 性能计时器
type Timer struct {
	start time.Time
	name  string
}

// NewTimer 创建计时器
func NewTimer(name string) *Timer {
	return &Timer{
		start: time.Now(),
		name:  name,
	}
}

// Elapsed 经过的时间
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// LogElapsed 记录经过的时间
func (t *Timer) LogElapsed() {
	duration := t.Elapsed()
	if t.name != "" {
		fmt.Printf("[TIMER] %s 耗时: %v\n", t.name, duration)
	} else {
		fmt.Printf("[TIMER] 耗时: %v\n", duration)
	}
}

// DeferLog 延迟记录
func (t *Timer) DeferLog() {
	defer t.LogElapsed()
}

// BatchProcessor 批处理器
type BatchProcessor[T any] struct {
	batchSize int
	timeout   time.Duration
	processor func([]T)
	batch     []T
	timer     *time.Timer
	mu        sync.Mutex
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor[T any](batchSize int, timeout time.Duration, processor func([]T)) *BatchProcessor[T] {
	if batchSize <= 0 {
		batchSize = 10
	}
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}
	bp := &BatchProcessor[T]{
		batchSize: batchSize,
		timeout:   timeout,
		processor: processor,
		batch:     make([]T, 0, batchSize),
		timer:     time.NewTimer(timeout),
	}
	go bp.process()
	return bp
}

// Add 添加项目
func (bp *BatchProcessor[T]) Add(item T) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.batch = append(bp.batch, item)
	if len(bp.batch) >= bp.batchSize {
		bp.flush()
	}
}

// process 处理批次
func (bp *BatchProcessor[T]) process() {
	for range bp.timer.C {
		bp.mu.Lock()
		if len(bp.batch) > 0 {
			bp.flush()
		}
		bp.mu.Unlock()
		bp.timer.Reset(bp.timeout)
	}
}

// flush 刷新批次
func (bp *BatchProcessor[T]) flush() {
	batch := bp.batch
	bp.batch = make([]T, 0, bp.batchSize)
	SafeGo(func() {
		bp.processor(batch)
	})
}

// Stop 停止批处理器
func (bp *BatchProcessor[T]) Stop() {
	bp.timer.Stop()
	bp.mu.Lock()
	if len(bp.batch) > 0 {
		bp.flush()
	}
	bp.mu.Unlock()
}
