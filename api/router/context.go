// Package router 提供路由和上下文管理功能
package router

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Context 执行上下文，用于在函数调用过程中传递和管理上下文信息
type Context struct {
	SessionID string         // 会话唯一标识符
	AgentID   string         // 调用者身份标识
	TraceID   string         // 分布式追踪ID
	StartTime time.Time      // 上下文创建时间
	Router    *Router        // 路由器引用
	Values    map[string]any // 上下文键值对存储
	mu        sync.RWMutex   // 读写互斥锁
	Timeout   time.Duration  // 执行超时时间
	cancel    context.CancelFunc
	CallDepth int    // 当前调用深度
	ParentID  string // 父调用ID
}

// NewContext 创建新的执行上下文
func NewContext(sessionID, agentID string, router *Router) *Context {
	context := &Context{
		SessionID: sessionID,
		AgentID:   agentID,
		TraceID:   generateTraceID(),
		StartTime: time.Now(),
		Router:    router,
		Timeout:   router.config.DefaultTimeout,
		CallDepth: 0,
	}
	context.Values = map[string]any{
		"router":  router,
		"context": context,
	}
	return context
}

// GetCallDepth 获取当前调用深度
func (c *Context) GetCallDepth() int {
	return c.CallDepth
}

// IncrementCallDepth 增加调用深度
func (c *Context) IncrementCallDepth() {
	c.CallDepth++
}

// DecrementCallDepth 减少调用深度
func (c *Context) DecrementCallDepth() {
	if c.CallDepth > 0 {
		c.CallDepth--
	}
}

// SetValue 设置上下文键值对
func (c *Context) SetValue(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Values == nil {
		c.Values = make(map[string]any)
	}
	c.Values[key] = value
}

// GetValue 获取上下文中的值
func (c *Context) GetValue(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Values == nil {
		return nil
	}
	return c.Values[key]
}

// DeleteValue 删除上下文中的键值对
func (c *Context) DeleteValue(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Values != nil {
		delete(c.Values, key)
	}
}

// LogError 记录错误日志
func (c *Context) LogError(message string, fields map[string]any) {
	fmt.Printf("[ERROR] %s: %v\n", message, fields)
}

// generateTraceID 生成唯一追踪ID
func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}
