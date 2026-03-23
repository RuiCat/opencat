package router

import (
	"sync"
	"time"
)

// 数据块类型
type BlockType int

const (
	BlockTypeUnsubscribe BlockType = iota - 1 // 取消订阅块类型
	BlockTypeCommand                          // 命令块类型
	BlockTypeResult                           // 结果块类型
	BlockTypeEvent                            // 事件块类型
	BlockTypeLog                              // 日志块类型
	BlockTypeError                            // 错误块类型
)

// 崩溃恢复处理器
type RecoveryHandler func(ctx *Context, block *DataBlock, panicValue any) *Result

// 事件处理器
type EventHandler func(blockType BlockType, data any)

// 数据统一
type DataBlock struct {
	ID        string         `json:"id"`        // 唯一标识符
	Type      BlockType      `json:"type"`      // 数据块类型
	Timestamp int64          `json:"timestamp"` // 时间戳(毫秒)
	Source    string         `json:"source"`    // 调用来源
	Target    string         `json:"target"`    // 目标函数
	Payload   map[string]any `json:"payload"`   // 有效载荷
	TraceID   string         `json:"trace_id"`  // 追踪链路ID
}

// 事件总线
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
}

// 函数定义
type Function struct {
	Name         string         `json:"name"`          // 函数名称
	Description  string         `json:"description"`   // 函数描述
	Namespace    string         `json:"namespace"`     // 命名空间
	InputSchema  []string       `json:"input_schema"`  // 参数值介绍
	OutputSchema []string       `json:"output_schema"` // 返回值介绍
	Function     any            `json:"-"`             // 函数实现
	IsMethod     bool           `json:"is_method"`     // 结构体函数
	Builtin      bool           `json:"builtin"`       // 是否内置函数
	Enabled      bool           `json:"enabled"`       // 是否启用
	CreatedAt    time.Time      `json:"created_at"`    // 创建时间
	Stats        *FunctionStats `json:"stats"`         // 统计信息
}

// FunctionStats 函数统计信息
type FunctionStats struct {
	CallCount    int64         `json:"call_count"`     // 总调用次数
	SuccessCount int64         `json:"success_count"`  // 成功次数
	FailureCount int64         `json:"failure_count"`  // 失败次数
	PanicCount   int64         `json:"panic_count"`    // panic次数
	AvgDuration  time.Duration `json:"avg_duration"`   // 平均执行时长
	LastCalledAt time.Time     `json:"last_called_at"` // 最后调用时间
}

// 错误详情
type ErrorInfo struct {
	Code      string `json:"code"`            // 错误码
	Message   string `json:"message"`         // 错误消息
	Stack     string `json:"stack,omitempty"` // 堆栈信息
	Recovered bool   `json:"recovered"`       // 是否从panic恢复
	Retryable bool   `json:"retryable"`       // 是否可重试
}

// 执行结果
type Result struct {
	Success  bool       `json:"success"`         // 是否成功
	Data     any        `json:"data,omitempty"`  // 返回数据
	Error    *ErrorInfo `json:"error,omitempty"` // 错误信息
	Duration int64      `json:"duration_ms"`     // 执行耗时(毫秒)
	TraceID  string     `json:"trace_id"`        // 追踪ID
}

// 路由器
type RouterConfig struct {
	MaxCallDepth       int                   // 最大调用深度
	DefaultTimeout     time.Duration         // 默认超时时间
	EnableAuditLog     bool                  // 是否启用审计日志
	EnableRecovery     bool                  // 是否启用崩溃恢复
	MaxFunctions       int                   // 最大函数数量
	MaxConcurrentCalls int                   // 最大并发调用数
	EnableTriggers     bool                  // 是否启用触发器
	EnableAsyncEvents  bool                  // 是否启用异步事件
	TriggerConfig      *TriggerManagerConfig // 触发器配置
}

// 路由器
type RouterStats struct {
	TotalCalls      int64         `json:"total_calls"`       // 总调用次数
	SuccessfulCalls int64         `json:"successful_calls"`  // 成功调用次数
	FailedCalls     int64         `json:"failed_calls"`      // 失败调用次数
	PanicCalls      int64         `json:"panic_calls"`       // panic调用次数
	AvgCallDuration time.Duration `json:"avg_call_duration"` // 平均调用时长
	StartTime       time.Time     `json:"start_time"`        // 启动时间
	FunctionCount   int           `json:"function_count"`    // 函数数量
}

// TriggerManager 触发器管理器
type TriggerManager struct {
	mu       sync.RWMutex
	triggers map[string]*Trigger
	eventBus *EventBus
	stats    *TriggerManagerStats
	config   *TriggerManagerConfig
}

// TriggerManagerStats 触发器管理器统计信息
type TriggerManagerStats struct {
	TotalTriggers   int       `json:"total_triggers"`   // 触发器总数
	EnabledTriggers int       `json:"enabled_triggers"` // 启用数量
	TotalEvents     int64     `json:"total_events"`     // 事件总数
	TriggeredCount  int64     `json:"triggered_count"`  // 触发次数
	SuccessCount    int64     `json:"success_count"`    // 成功次数
	ErrorCount      int64     `json:"error_count"`      // 失败次数
	StartTime       time.Time `json:"start_time"`       // 启动时间
	LastEventTime   time.Time `json:"last_event_time"`  // 最后事件时间
}

// TriggerManagerConfig 触发器管理器配置
type TriggerManagerConfig struct {
	MaxTriggers        int  `json:"max_triggers"`         // 最大触发器数量
	EnableAsync        bool `json:"enable_async"`         // 是否异步执行
	MaxConcurrentFires int  `json:"max_concurrent_fires"` // 最大并发触发数
	EventBufferSize    int  `json:"event_buffer_size"`    // 事件缓冲区大小
	EnableStats        bool `json:"enable_stats"`         // 是否启用统计
}

// 触发定义
type Trigger struct {
	ID           string                   `json:"id"`            // 唯一标识符
	Name         string                   `json:"name"`          // 触发器名称
	Description  string                   `json:"description"`   // 功能描述
	EventPattern string                   `json:"event_pattern"` // 事件匹配模式(支持通配符)
	Condition    func(event *Event) bool  `json:"-"`             // 条件检查函数
	Action       func(event *Event) error `json:"-"`             // 执行动作
	Enabled      bool                     `json:"enabled"`       // 是否启用
	Priority     int                      `json:"priority"`      // 优先级(越高越先执行)
	CreatedAt    time.Time                `json:"created_at"`    // 创建时间
	LastFired    time.Time                `json:"last_fired"`    // 上次触发时间
	FireCount    int                      `json:"fire_count"`    // 累计触发次数
	SuccessCount int                      `json:"success_count"` // 成功次数
	ErrorCount   int                      `json:"error_count"`   // 失败次数
	LastError    string                   `json:"last_error"`    // 最后错误信息
}

// 事件定义
type Event struct {
	Name    string         `json:"name"`     // 事件名称
	Source  string         `json:"source"`   // 事件来源
	Data    map[string]any `json:"data"`     // 事件数据
	Time    time.Time      `json:"time"`     // 发生时间
	TraceID string         `json:"trace_id"` // 追踪ID
}
