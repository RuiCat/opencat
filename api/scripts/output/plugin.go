// 输出函数插件
// 这个插件提供了标准化的输出函数，可以替换fmt.Println等调用

package output

import (
	"context"
	"fmt"
	"time"

	"api/plugin"
)

// PluginInfo 插件信息
var PluginInfo = plugin.PluginInfo{
	Name:         "输出函数插件",
	Version:      "0.0.1",
	Description:  "提供标准化的输出函数，支持不同级别的日志输出",
	Author:       "OpenClaw System",
	Dependencies: []string{},
	Metadata: map[string]string{
		"category": "utility",
		"type":     "output",
		"tags":     "log,output,print,debug",
	},
}

// OutputLevel 输出级别
type OutputLevel int

const (
	LevelDebug OutputLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// OutputPlugin 输出插件
type OutputPlugin struct {
	name    string
	enabled bool
	config  *OutputConfig
	tools   map[string]plugin.ToolHandler
	events  map[string][]plugin.EventHandler
}

// OutputConfig 输出配置
type OutputConfig struct {
	MinLevel   OutputLevel `json:"min_level"`
	Timestamp  bool        `json:"timestamp"`
	LevelLabel bool        `json:"level_label"`
	Source     bool        `json:"source"`
	Color      bool        `json:"color"`
}

// NewOutputPlugin 创建新的输出插件
func NewOutputPlugin() *OutputPlugin {
	return &OutputPlugin{
		name:    "output",
		enabled: true,
		config: &OutputConfig{
			MinLevel:   LevelInfo,
			Timestamp:  true,
			LevelLabel: true,
			Source:     false,
			Color:      true,
		},
		tools:  make(map[string]plugin.ToolHandler),
		events: make(map[string][]plugin.EventHandler),
	}
}

// ============================================================================
// 插件接口实现
// ============================================================================

// GetInfo 获取插件信息
func (p *OutputPlugin) GetInfo() *plugin.PluginInfo {
	return &PluginInfo
}

// Init 初始化插件
func (p *OutputPlugin) Init(ctx context.Context) error {
	p.Println("输出插件初始化...")

	// 注册工具
	p.registerTools()

	// 订阅事件
	p.subscribeEvents()

	p.Println("输出插件初始化完成")
	return nil
}

// Shutdown 关闭插件
func (p *OutputPlugin) Shutdown(ctx context.Context) error {
	p.Println("输出插件关闭...")
	p.enabled = false
	p.tools = make(map[string]plugin.ToolHandler)
	p.events = make(map[string][]plugin.EventHandler)
	return nil
}

// Enable 启用插件
func (p *OutputPlugin) Enable() error {
	p.enabled = true
	return nil
}

// Disable 禁用插件
func (p *OutputPlugin) Disable() error {
	p.enabled = false
	return nil
}

// IsEnabled 检查插件是否启用
func (p *OutputPlugin) IsEnabled() bool {
	return p.enabled
}

// RegisterTool 注册工具
func (p *OutputPlugin) RegisterTool(tool plugin.ToolDefinition, handler plugin.ToolHandler) error {
	p.tools[tool.Name] = handler
	return nil
}

// UnregisterTool 注销工具
func (p *OutputPlugin) UnregisterTool(name string) error {
	delete(p.tools, name)
	return nil
}

// ListTools 列出工具
func (p *OutputPlugin) ListTools() []plugin.ToolDefinition {
	tools := make([]plugin.ToolDefinition, 0, len(p.tools))
	for name := range p.tools {
		tools = append(tools, plugin.ToolDefinition{
			Name:        name,
			Description: "输出工具",
			Enabled:     true,
		})
	}
	return tools
}

// CallTool 调用工具
func (p *OutputPlugin) CallTool(ctx context.Context, name string, args map[string]interface{}) (*plugin.ToolResult, error) {
	handler, exists := p.tools[name]
	if !exists {
		return nil, plugin.ErrToolNotFound
	}

	if !p.enabled {
		return &plugin.ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	return handler(ctx, args)
}

// EmitEvent 发送事件
func (p *OutputPlugin) EmitEvent(ctx context.Context, event *plugin.Event) error {
	p.Printf("输出插件发送事件: %s\n", event.Type)
	return nil
}

// SubscribeEvent 订阅事件
func (p *OutputPlugin) SubscribeEvent(eventType string, handler plugin.EventHandler) error {
	p.events[eventType] = append(p.events[eventType], handler)
	return nil
}

// UnsubscribeEvent 取消订阅事件
func (p *OutputPlugin) UnsubscribeEvent(eventType string, handler plugin.EventHandler) error {
	handlers, exists := p.events[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if &h == &handler {
			p.events[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// CheckDependencies 检查插件依赖
func (p *OutputPlugin) CheckDependencies(ctx context.Context) error {
	return nil
}

// GetDependencies 获取插件依赖
func (p *OutputPlugin) GetDependencies() []string {
	return PluginInfo.Dependencies
}

// ============================================================================
// 工具注册
// ============================================================================

// registerTools 注册工具
func (p *OutputPlugin) registerTools() {
	// 注册Println工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "println",
		Description: "输出一行文本",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
				"level": map[string]interface{}{
					"type": "string",
					"enum": []string{"debug", "info", "warn", "error", "fatal"},
				},
			},
			Required: []string{"message"},
		},
		Enabled: true,
	}, p.printlnToolHandler)

	// 注册Printf工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "printf",
		Description: "格式化输出文本",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"format": map[string]interface{}{
					"type": "string",
				},
				"args": map[string]interface{}{
					"type": "array",
				},
				"level": map[string]interface{}{
					"type": "string",
					"enum": []string{"debug", "info", "warn", "error", "fatal"},
				},
			},
			Required: []string{"format"},
		},
		Enabled: true,
	}, p.printfToolHandler)

	// 注册配置工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "configure",
		Description: "配置输出参数",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"min_level": map[string]interface{}{
					"type": "string",
					"enum": []string{"debug", "info", "warn", "error", "fatal"},
				},
				"timestamp": map[string]interface{}{
					"type": "boolean",
				},
				"level_label": map[string]interface{}{
					"type": "boolean",
				},
				"source": map[string]interface{}{
					"type": "boolean",
				},
				"color": map[string]interface{}{
					"type": "boolean",
				},
			},
		},
		Enabled: true,
	}, p.configureToolHandler)

	// 注册状态工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "status",
		Description: "检查插件状态",
		Enabled:     true,
	}, p.statusToolHandler)
}

// ============================================================================
// 工具处理器
// ============================================================================

// printlnToolHandler println工具处理器
func (p *OutputPlugin) printlnToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	message, ok := args["message"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "参数错误: message必须是字符串",
		}, nil
	}

	levelStr, _ := args["level"].(string)
	level := p.parseLevel(levelStr)

	// 检查级别过滤
	if level < p.config.MinLevel {
		return &plugin.ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"filtered": true,
				"level":    levelStr,
				"message":  message,
			},
		}, nil
	}

	// 实际输出
	p.output(level, message)

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"output":    true,
			"level":     levelStr,
			"message":   message,
			"timestamp": time.Now().Unix(),
		},
	}, nil
}

// printfToolHandler printf工具处理器
func (p *OutputPlugin) printfToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	format, ok := args["format"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "参数错误: format必须是字符串",
		}, nil
	}

	// 解析参数
	var formatArgs []interface{}
	if argsRaw, ok := args["args"].([]interface{}); ok {
		formatArgs = argsRaw
	}

	levelStr, _ := args["level"].(string)
	level := p.parseLevel(levelStr)

	// 检查级别过滤
	if level < p.config.MinLevel {
		return &plugin.ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"filtered": true,
				"level":    levelStr,
				"format":   format,
				"args":     formatArgs,
			},
		}, nil
	}

	// 格式化消息
	message := fmt.Sprintf(format, formatArgs...)

	// 实际输出
	p.output(level, message)

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"output":    true,
			"level":     levelStr,
			"message":   message,
			"format":    format,
			"args":      formatArgs,
			"timestamp": time.Now().Unix(),
		},
	}, nil
}

// configureToolHandler configure工具处理器
func (p *OutputPlugin) configureToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	updated := false

	// 更新min_level
	if minLevel, ok := args["min_level"].(string); ok && minLevel != "" {
		p.config.MinLevel = p.parseLevel(minLevel)
		updated = true
	}

	// 更新timestamp
	if timestamp, ok := args["timestamp"].(bool); ok {
		p.config.Timestamp = timestamp
		updated = true
	}

	// 更新level_label
	if levelLabel, ok := args["level_label"].(bool); ok {
		p.config.LevelLabel = levelLabel
		updated = true
	}

	// 更新source
	if source, ok := args["source"].(bool); ok {
		p.config.Source = source
		updated = true
	}

	// 更新color
	if color, ok := args["color"].(bool); ok {
		p.config.Color = color
		updated = true
	}

	if updated {
		// 发送配置更新事件
		p.EmitEvent(ctx, &plugin.Event{
			Type:      "output.config.updated",
			Name:      "config_updated",
			Data:      p.config,
			Timestamp: time.Now(),
			Source:    "output_plugin",
		})

		return &plugin.ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"message": "配置更新成功",
				"config":  p.config,
			},
		}, nil
	}

	return &plugin.ToolResult{
		Success: false,
		Error:   "没有提供有效的配置参数",
	}, nil
}

// statusToolHandler status工具处理器
func (p *OutputPlugin) statusToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	status := "disabled"
	if p.enabled {
		status = "enabled"
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"status":      status,
			"enabled":     p.enabled,
			"tools_count": len(p.tools),
			"config":      p.config,
		},
	}, nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// parseLevel 解析级别字符串
func (p *OutputPlugin) parseLevel(levelStr string) OutputLevel {
	switch levelStr {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// output 实际输出方法
func (p *OutputPlugin) output(level OutputLevel, message string) {
	// 构建输出前缀
	var prefix string

	// 添加时间戳
	if p.config.Timestamp {
		prefix += time.Now().Format("2006-01-02 15:04:05 ") + " "
	}

	// 添加级别标签
	if p.config.LevelLabel {
		switch level {
		case LevelDebug:
			prefix += "[DEBUG] "
		case LevelInfo:
			prefix += "[INFO] "
		case LevelWarn:
			prefix += "[WARN] "
		case LevelError:
			prefix += "[ERROR] "
		case LevelFatal:
			prefix += "[FATAL] "
		}
	}

	// 实际输出到标准输出
	fmt.Println(prefix + message)
}

// subscribeEvents 订阅事件
func (p *OutputPlugin) subscribeEvents() {
	// 订阅系统事件
	p.SubscribeEvent("system.startup", func(event *plugin.Event) error {
		p.Println("输出插件收到系统启动事件")
		return nil
	})

	p.SubscribeEvent("system.shutdown", func(event *plugin.Event) error {
		p.Println("输出插件收到系统关闭事件")
		return nil
	})
}

// ============================================================================
// 导出函数（供其他插件调用）
// ============================================================================

// Println 输出一行文本
func (p *OutputPlugin) Println(message string) {
	p.output(LevelInfo, message)
}

// Printf 格式化输出文本
func (p *OutputPlugin) Printf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	p.output(LevelInfo, message)
}

// Debug 调试级别输出
func (p *OutputPlugin) Debug(message string) {
	p.output(LevelDebug, message)
}

// Info 信息级别输出
func (p *OutputPlugin) Info(message string) {
	p.output(LevelInfo, message)
}

// Warn 警告级别输出
func (p *OutputPlugin) Warn(message string) {
	p.output(LevelWarn, message)
}

// Error 错误级别输出
func (p *OutputPlugin) Error(message string) {
	p.output(LevelError, message)
}

// ============================================================================
// 插件导出函数
// ============================================================================

// NewPlugin 创建插件实例（供插件管理器调用）
func NewPlugin() plugin.Plugin {
	return NewOutputPlugin()
}

// GetPluginInfo 获取插件信息（供Yaegi调用）
func GetPluginInfo() *plugin.PluginInfo {
	return &PluginInfo
}
