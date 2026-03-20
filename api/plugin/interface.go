package plugin

import (
	"context"
	"time"
)

// ============================================================================
// 插件核心接口定义
// ============================================================================

// PluginInfo 插件元数据
type PluginInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author,omitempty"`
	Homepage     string            `json:"homepage,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"` // 依赖的函数名列表
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at,omitempty"`
}

// ============================================================================
// 工具系统接口
// ============================================================================

// ToolSchema 工具输入模式定义
type ToolSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema *ToolSchema `json:"inputSchema,omitempty"`
	Enabled     bool        `json:"enabled"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// ToolHandler 工具处理器函数签名
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)

// ============================================================================
// 事件系统接口
// ============================================================================

// Event 事件
type Event struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
}

// EventHandler 事件处理器
type EventHandler func(event *Event) error

// ============================================================================
// 插件生命周期接口
// ============================================================================

// Plugin 插件接口
type Plugin interface {
	// 插件信息
	GetInfo() *PluginInfo

	// 生命周期管理
	Init(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Enable() error
	Disable() error
	IsEnabled() bool

	// 工具管理
	RegisterTool(tool ToolDefinition, handler ToolHandler) error
	UnregisterTool(name string) error
	ListTools() []ToolDefinition
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)

	// 事件管理
	EmitEvent(ctx context.Context, event *Event) error
	SubscribeEvent(eventType string, handler EventHandler) error
	UnsubscribeEvent(eventType string, handler EventHandler) error

	// 依赖管理
	CheckDependencies(ctx context.Context) error
	GetDependencies() []string
}

// ============================================================================
// 插件管理器接口
// ============================================================================

// PluginManagerInterface 插件管理器接口
type PluginManagerInterface interface {
	// 插件管理
	LoadPlugin(name string, plugin Plugin) error
	UnloadPlugin(name string) error
	GetPlugin(name string) (Plugin, error)
	ListPlugins() []PluginInfo

	// 工具管理
	RegisterTool(pluginName string, tool ToolDefinition, handler ToolHandler) error
	UnregisterTool(pluginName, toolName string) error
	ListTools() []ToolDefinition
	CallTool(ctx context.Context, pluginName, toolName string, args map[string]interface{}) (*ToolResult, error)

	// 事件管理
	EmitEvent(ctx context.Context, event *Event) error
	SubscribeEvent(eventType string, handler EventHandler) error
	UnsubscribeEvent(eventType string, handler EventHandler) error

	// 状态管理
	EnablePlugin(name string) error
	DisablePlugin(name string) error
	GetPluginStatus(name string) (bool, error)

	// 依赖管理
	CheckPluginDependencies(name string) error
	GetPluginDependencies(name string) ([]string, error)
}

// ============================================================================
// 路由接口（插件调用路由）
// ============================================================================

// Router 路由接口
type Router interface {
	// 函数管理
	RegisterFunction(name string, handler interface{}) error
	UnregisterFunction(name string) error
	CallFunction(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
	ListFunctions() []string
	HasFunction(name string) bool

	// 事件管理
	PublishEvent(eventType string, data interface{}) error
	SubscribeEvent(eventType string, handler interface{}) error

	// 插件管理
	RegisterPlugin(plugin Plugin) error
	UnregisterPlugin(name string) error
	GetPluginManager() PluginManagerInterface

	// 管理接口注册
	RegisterManagementInterface(name string, handler interface{}) error
	GetManagementInterface(name string) (interface{}, error)
}

// ============================================================================
// 函数调用接口（脚本调用其他脚本）
// ============================================================================

// FunctionCaller 函数调用接口
type FunctionCaller interface {
	// 调用函数
	Call(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)

	// 检查函数是否存在
	Has(name string) bool

	// 列出所有可用函数
	List() []string

	// 获取函数信息
	GetInfo(name string) (map[string]interface{}, error)
}

// ============================================================================
// 错误类型定义
// ============================================================================

// PluginError 插件错误
type PluginError struct {
	PluginName string
	Message    string
	Cause      error
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return e.PluginName + ": " + e.Message + " - " + e.Cause.Error()
	}
	return e.PluginName + ": " + e.Message
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}

// 预定义错误
var (
	ErrPluginNotFound        = &PluginError{Message: "插件未找到"}
	ErrPluginAlreadyLoaded   = &PluginError{Message: "插件已加载"}
	ErrPluginNotLoaded       = &PluginError{Message: "插件未加载"}
	ErrPluginInitFailed      = &PluginError{Message: "插件初始化失败"}
	ErrPluginShutdownFailed  = &PluginError{Message: "插件关闭失败"}
	ErrToolNotFound          = &PluginError{Message: "工具未找到"}
	ErrToolAlreadyRegistered = &PluginError{Message: "工具已注册"}
	ErrPluginSyntaxError     = &PluginError{Message: "插件语法错误"}
	ErrPluginDisabled        = &PluginError{Message: "插件已禁用"}
	ErrInvalidPlugin         = &PluginError{Message: "无效的插件"}
	ErrDependencyMissing     = &PluginError{Message: "依赖缺失"}
	ErrCircularDependency    = &PluginError{Message: "循环依赖"}
)
