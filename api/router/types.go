package router

import (
	"time"
)

// BlockType 数据块类型
type BlockType int

const (
	BlockTypeCommand BlockType = iota // 命令调用
	BlockTypeResult                   // 执行结果
	BlockTypeEvent                    // 事件通知
	BlockTypeLog                      // 日志记录
	BlockTypeError                    // 错误信息
)

// DataBlock 统一数据块
type DataBlock struct {
	ID        string                 `json:"id"`
	Type      BlockType              `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Source    string                 `json:"source"`    // 调用来源
	Target    string                 `json:"target"`    // 目标函数
	Payload   map[string]interface{} `json:"payload"`   // 参数
	Metadata  map[string]string      `json:"metadata"`  // 元数据
	TraceID   string                 `json:"trace_id"`  // 追踪链路
	ParentID  string                 `json:"parent_id"` // 父调用ID（用于调用链）
}

// NewDataBlock 创建新的数据块
func NewDataBlock(target string, payload map[string]interface{}) *DataBlock {
	return &DataBlock{
		ID:        generateID(),
		Type:      BlockTypeCommand,
		Timestamp: time.Now().UnixNano(),
		Target:    target,
		Payload:   payload,
		Metadata:  make(map[string]string),
		TraceID:   generateTraceID(),
	}
}

// Result 执行结果
type Result struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data,omitempty"`
	Error    *ErrorInfo  `json:"error,omitempty"`
	Duration int64       `json:"duration_ms"`
	TraceID  string      `json:"trace_id"`
	Logs     []LogEntry  `json:"logs,omitempty"` // 执行过程日志
}

// ErrorInfo 错误详情
type ErrorInfo struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Stack     string `json:"stack,omitempty"` // 堆栈信息
	Recovered bool   `json:"recovered"`       // 是否从 panic 恢复
	Retryable bool   `json:"retryable"`       // 是否可重试
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Function 函数定义
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Namespace   string                 `json:"namespace"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Handler     FunctionHandler        `json:"-"`
	Builtin     bool                   `json:"builtin"` // 是否内置（不可删除）
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	Stats       *FunctionStats         `json:"stats"`
}

// FunctionStats 函数统计
type FunctionStats struct {
	CallCount    int64         `json:"call_count"`
	SuccessCount int64         `json:"success_count"`
	FailureCount int64         `json:"failure_count"`
	PanicCount   int64         `json:"panic_count"`
	AvgDuration  time.Duration `json:"avg_duration"`
	LastCalledAt time.Time     `json:"last_called_at"`
}

// FunctionHandler 函数处理器
type FunctionHandler func(ctx *Context, block *DataBlock) *Result

// RecoveryHandler 崩溃恢复处理器
type RecoveryHandler func(ctx *Context, block *DataBlock, panicValue interface{}) *Result

// Interceptor 拦截器
type Interceptor interface {
	Before(ctx *Context, block *DataBlock) error
	After(ctx *Context, block *DataBlock, result *Result)
}

// EventHandler 事件处理器
type EventHandler func(event string, data interface{})

// 辅助函数
func generateID() string {
	return "block_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

func generateTraceID() string {
	return "trace_" + time.Now().Format("20060102150405") + "_" + randomString(12)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
