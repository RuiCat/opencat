package router

import (
	"sync"
	"time"
)

// Interceptor 拦截器接口
type Interceptor interface {
	Intercept(ctx *Context, call *InterceptorCall) error // 拦截器执行，返回错误将中断链
}

// InterceptorCall 拦截器调用
type InterceptorCall struct {
	Name     string         // 函数名称
	Args     []any          // 调用参数
	Result   any            // 调用结果
	Err      error          // 调用错误
	Start    time.Time      // 开始时间
	Duration time.Duration  // 执行时长
	Metadata map[string]any // 元数据
}

// InterceptorChain 拦截器链
type InterceptorChain struct {
	head *interceptorNode
	tail *interceptorNode
	mu   sync.RWMutex
}

type interceptorNode struct {
	interceptor Interceptor
	next        *interceptorNode
	priority    int // 优先级，越小越先执行
}

// NewInterceptorChain 创建新的拦截器链
func NewInterceptorChain() *InterceptorChain {
	return &InterceptorChain{}
}

// Add 添加拦截器
func (c *InterceptorChain) Add(interceptor Interceptor, priority int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node := &interceptorNode{
		interceptor: interceptor,
		priority:    priority,
	}
	if c.head == nil {
		c.head = node
		c.tail = node
		return
	}
	if node.priority < c.head.priority {
		node.next = c.head
		c.head = node
		return
	}
	current := c.head
	for current.next != nil && current.next.priority <= node.priority {
		current = current.next
	}
	node.next = current.next
	current.next = node
	if node.next == nil {
		c.tail = node
	}
}

// Remove 移除拦截器
func (c *InterceptorChain) Remove(interceptor Interceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.head == nil {
		return
	}
	if c.head.interceptor == interceptor {
		c.head = c.head.next
		if c.head == nil {
			c.tail = nil
		}
		return
	}
	prev := c.head
	current := c.head.next
	for current != nil {
		if current.interceptor == interceptor {
			prev.next = current.next
			if current == c.tail {
				c.tail = prev
			}
			return
		}
		prev = current
		current = current.next
	}
}

// Intercept 拦截器执行
func (c *InterceptorChain) Intercept(ctx *Context, call *InterceptorCall) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	current := c.head
	for current != nil {
		if err := current.interceptor.Intercept(ctx, call); err != nil {
			return err
		}
		current = current.next
	}
	return nil
}

// IsEmpty 检查是否为空
func (c *InterceptorChain) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.head == nil
}

// Clear 清空链
func (c *InterceptorChain) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.head = nil
	c.tail = nil
}

// Count 拦截器数量
func (c *InterceptorChain) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	current := c.head
	for current != nil {
		count++
		current = current.next
	}
	return count
}

// NewInterceptorCall 创建新的拦截器调用
func NewInterceptorCall(name string, args []any) *InterceptorCall {
	return &InterceptorCall{
		Name:     name,
		Args:     args,
		Start:    time.Now(),
		Metadata: make(map[string]any),
	}
}

// Complete 完成调用并记录结果
func (c *InterceptorCall) Complete(result any, err error) {
	c.Result = result
	c.Err = err
	c.Duration = time.Since(c.Start)
}
