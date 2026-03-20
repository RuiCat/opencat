package router

import (
	"context"
	"sync"
	"time"
)

// Context 执行上下文
type Context struct {
	// 基础信息
	SessionID string
	AgentID   string // 调用者身份
	TraceID   string // 追踪ID
	StartTime time.Time

	// 路由引用
	Router *Router // 路由引用（允许函数内调用其他函数）

	// 上下文数据
	Values map[string]interface{} // 上下文数据
	mu     sync.RWMutex           // 保护 Values 的锁

	// 控制
	Timeout time.Duration
	cancel  context.CancelFunc

	// 日志记录器
	Logger *Logger

	// 调用链信息
	CallDepth int    // 当前调用深度
	ParentID  string // 父调用ID
}

// NewContext 创建新的执行上下文
func NewContext(sessionID, agentID string, router *Router) *Context {
	ctx := &Context{
		SessionID: sessionID,
		AgentID:   agentID,
		Router:    router,
		StartTime: time.Now(),
		Values:    make(map[string]interface{}),
		Logger:    NewLogger(),
		CallDepth: 0,
		TraceID:   generateTraceID(),
	}

	// 设置默认超时
	ctx.Timeout = 30 * time.Second

	return ctx
}

// WithTimeout 设置超时
func (c *Context) WithTimeout(timeout time.Duration) *Context {
	c.Timeout = timeout
	return c
}

// WithValue 设置上下文值
func (c *Context) WithValue(key string, value interface{}) *Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Values[key] = value
	return c
}

// Value 获取上下文值
func (c *Context) Value(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Values[key]
}

// DeleteValue 删除上下文值
func (c *Context) DeleteValue(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Values, key)
}

// IncCallDepth 增加调用深度
func (c *Context) IncCallDepth() *Context {
	c.CallDepth++
	return c
}

// DecCallDepth 减少调用深度
func (c *Context) DecCallDepth() *Context {
	if c.CallDepth > 0 {
		c.CallDepth--
	}
	return c
}

// GetCallDepth 获取调用深度
func (c *Context) GetCallDepth() int {
	return c.CallDepth
}

// SetParentID 设置父调用ID
func (c *Context) SetParentID(parentID string) *Context {
	c.ParentID = parentID
	return c
}

// GetParentID 获取父调用ID
func (c *Context) GetParentID() string {
	return c.ParentID
}

// Log 记录日志
func (c *Context) Log(level, message string, fields map[string]interface{}) {
	if c.Logger != nil {
		c.Logger.Log(level, message, fields)
	}
}

// LogInfo 记录信息日志
func (c *Context) LogInfo(message string, fields map[string]interface{}) {
	c.Log("info", message, fields)
}

// LogError 记录错误日志
func (c *Context) LogError(message string, fields map[string]interface{}) {
	c.Log("error", message, fields)
}

// LogDebug 记录调试日志
func (c *Context) LogDebug(message string, fields map[string]interface{}) {
	c.Log("debug", message, fields)
}

// GetElapsedTime 获取已用时间
func (c *Context) GetElapsedTime() time.Duration {
	return time.Since(c.StartTime)
}

// IsTimeout 检查是否超时
func (c *Context) IsTimeout() bool {
	if c.Timeout <= 0 {
		return false
	}
	return c.GetElapsedTime() > c.Timeout
}

// Cancel 取消上下文
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Clone 克隆上下文（用于子调用）
func (c *Context) Clone() *Context {
	clone := &Context{
		SessionID: c.SessionID,
		AgentID:   c.AgentID,
		Router:    c.Router,
		StartTime: time.Now(),
		Values:    make(map[string]interface{}),
		Logger:    c.Logger,
		CallDepth: c.CallDepth + 1,
		TraceID:   c.TraceID,
		ParentID:  c.TraceID, // 父调用ID设置为当前TraceID
		Timeout:   c.Timeout,
	}

	// 复制 Values
	c.mu.RLock()
	for k, v := range c.Values {
		clone.Values[k] = v
	}
	c.mu.RUnlock()

	return clone
}
