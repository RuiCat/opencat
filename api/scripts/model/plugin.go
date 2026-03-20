// 模型提供商API调用插件
// 这个插件支持多种模型API提供商：SiliconFlow、OpenAI、DeepSeek、DashScope等

package model

import (
	"context"
	"fmt"
	"time"

	"api/plugin"
)

// PluginInfo 插件信息
var PluginInfo = plugin.PluginInfo{
	Name:         "模型提供商API调用",
	Version:      "0.0.1",
	Description:  "支持多种模型API提供商的调用插件，包括SiliconFlow、OpenAI、DeepSeek、DashScope等",
	Author:       "OpenClaw System",
	Dependencies: []string{},
	Metadata: map[string]string{
		"category": "model",
		"type":     "api",
		"tags":     "ai,model,api,chat,provider",
	},
}

// ModelProvider 模型提供商类型
type ModelProvider string

const (
	ProviderSiliconFlow ModelProvider = "siliconflow"
	ProviderOpenAI      ModelProvider = "openai"
	ProviderDeepSeek    ModelProvider = "deepseek"
	ProviderDashScope   ModelProvider = "dashscope"
	ProviderCustom      ModelProvider = "custom"
)

// OutputInterface 输出接口
type OutputInterface interface {
	Println(message string)
	Printf(format string, args ...interface{})
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}

// ModelAPIPlugin 模型API插件
type ModelAPIPlugin struct {
	name     string
	enabled  bool
	config   *PluginConfig
	tools    map[string]plugin.ToolHandler
	events   map[string][]plugin.EventHandler
	provider ModelProvider
	clients  map[ModelProvider]ModelClient
	output   OutputInterface // 输出接口
}

// PluginConfig 插件配置
type PluginConfig struct {
	DefaultProvider ModelProvider             `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Timeout int    `json:"timeout"`
}

// ModelClient 模型客户端接口
type ModelClient interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	TestConnection(ctx context.Context) error
	GetProvider() ModelProvider
}

// ChatRequest 对话请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Provider    string    `json:"provider,omitempty"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Provider ModelProvider `json:"provider"`
}

// NewModelAPIPlugin 创建新的模型API插件
func NewModelAPIPlugin() *ModelAPIPlugin {
	return &ModelAPIPlugin{
		name:    "model_api",
		enabled: true,
		config: &PluginConfig{
			DefaultProvider: ProviderSiliconFlow,
			Providers: map[string]ProviderConfig{
				string(ProviderSiliconFlow): {
					BaseURL: "https://api.siliconflow.cn/v1",
					Model:   "deepseek-ai/DeepSeek-V3",
					Timeout: 120,
				},
				string(ProviderOpenAI): {
					BaseURL: "https://api.openai.com/v1",
					Model:   "gpt-3.5-turbo",
					Timeout: 120,
				},
				string(ProviderDeepSeek): {
					BaseURL: "https://api.deepseek.com/v1",
					Model:   "deepseek-chat",
					Timeout: 120,
				},
				string(ProviderDashScope): {
					BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
					Model:   "qwen-max",
					Timeout: 120,
				},
			},
		},
		tools:    make(map[string]plugin.ToolHandler),
		events:   make(map[string][]plugin.EventHandler),
		clients:  make(map[ModelProvider]ModelClient),
		provider: ProviderSiliconFlow,
	}
}

// ============================================================================
// 插件接口实现
// ============================================================================

// GetInfo 获取插件信息
func (p *ModelAPIPlugin) GetInfo() *plugin.PluginInfo {
	return &PluginInfo
}

// Init 初始化插件
func (p *ModelAPIPlugin) Init(ctx context.Context) error {
	// 初始化输出接口（如果可用）
	if p.output != nil {
		p.output.Info("模型API插件初始化...")
	} else {
		fmt.Println("模型API插件初始化...")
	}

	// 注册工具
	p.registerTools()

	// 初始化客户端
	p.initClients()

	// 订阅事件
	p.subscribeEvents()

	if p.output != nil {
		p.output.Info("模型API插件初始化完成")
	} else {
		fmt.Println("模型API插件初始化完成")
	}
	return nil
}

// Shutdown 关闭插件
func (p *ModelAPIPlugin) Shutdown(ctx context.Context) error {
	if p.output != nil {
		p.output.Info("模型API插件关闭...")
	} else {
		fmt.Println("模型API插件关闭...")
	}
	p.enabled = false
	p.tools = make(map[string]plugin.ToolHandler)
	p.events = make(map[string][]plugin.EventHandler)
	p.clients = make(map[ModelProvider]ModelClient)
	return nil
}

// Enable 启用插件
func (p *ModelAPIPlugin) Enable() error {
	p.enabled = true
	return nil
}

// Disable 禁用插件
func (p *ModelAPIPlugin) Disable() error {
	p.enabled = false
	return nil
}

// IsEnabled 检查插件是否启用
func (p *ModelAPIPlugin) IsEnabled() bool {
	return p.enabled
}

// RegisterTool 注册工具
func (p *ModelAPIPlugin) RegisterTool(tool plugin.ToolDefinition, handler plugin.ToolHandler) error {
	p.tools[tool.Name] = handler
	return nil
}

// UnregisterTool 注销工具
func (p *ModelAPIPlugin) UnregisterTool(name string) error {
	delete(p.tools, name)
	return nil
}

// ListTools 列出工具
func (p *ModelAPIPlugin) ListTools() []plugin.ToolDefinition {
	tools := make([]plugin.ToolDefinition, 0, len(p.tools))
	for name := range p.tools {
		tools = append(tools, plugin.ToolDefinition{
			Name:        name,
			Description: "模型API工具",
			Enabled:     true,
		})
	}
	return tools
}

// CallTool 调用工具
func (p *ModelAPIPlugin) CallTool(ctx context.Context, name string, args map[string]interface{}) (*plugin.ToolResult, error) {
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
func (p *ModelAPIPlugin) EmitEvent(ctx context.Context, event *plugin.Event) error {
	if p.output != nil {
		p.output.Printf("模型API插件发送事件: %s", event.Type)
	} else {
		fmt.Printf("模型API插件发送事件: %s\n", event.Type)
	}
	return nil
}

// SubscribeEvent 订阅事件
func (p *ModelAPIPlugin) SubscribeEvent(eventType string, handler plugin.EventHandler) error {
	p.events[eventType] = append(p.events[eventType], handler)
	return nil
}

// UnsubscribeEvent 取消订阅事件
func (p *ModelAPIPlugin) UnsubscribeEvent(eventType string, handler plugin.EventHandler) error {
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
func (p *ModelAPIPlugin) CheckDependencies(ctx context.Context) error {
	// 检查网络连接等依赖
	return nil
}

// GetDependencies 获取插件依赖
func (p *ModelAPIPlugin) GetDependencies() []string {
	return PluginInfo.Dependencies
}

// ============================================================================
// 工具注册
// ============================================================================

// registerTools 注册工具
func (p *ModelAPIPlugin) registerTools() {
	// 注册chat工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "chat",
		Description: "与AI模型对话，支持多种提供商",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"messages": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"role": map[string]interface{}{
								"type": "string",
								"enum": []string{"system", "user", "assistant"},
							},
							"content": map[string]interface{}{
								"type": "string",
							},
						},
						"required": []string{"role", "content"},
					},
				},
				"provider": map[string]interface{}{
					"type": "string",
					"enum": []string{"siliconflow", "openai", "deepseek", "dashscope"},
				},
				"model": map[string]interface{}{
					"type": "string",
				},
				"temperature": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 2.0,
				},
				"max_tokens": map[string]interface{}{
					"type":    "integer",
					"minimum": 1,
					"maximum": 4096,
				},
			},
			Required: []string{"messages"},
		},
		Enabled: true,
	}, p.chatToolHandler)

	// 注册配置工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "configure",
		Description: "配置模型API参数",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"provider": map[string]interface{}{
					"type": "string",
					"enum": []string{"siliconflow", "openai", "deepseek", "dashscope"},
				},
				"api_key": map[string]interface{}{
					"type": "string",
				},
				"model": map[string]interface{}{
					"type": "string",
				},
				"base_url": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Enabled: true,
	}, p.configureToolHandler)

	// 注册状态检查工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "status",
		Description: "检查插件状态和提供商配置",
		Enabled:     true,
	}, p.statusToolHandler)

	// 注册提供商列表工具
	p.RegisterTool(plugin.ToolDefinition{
		Name:        "providers",
		Description: "列出支持的模型提供商",
		Enabled:     true,
	}, p.providersToolHandler)
}

// ============================================================================
// 工具处理器
// ============================================================================

// chatToolHandler chat工具处理器
func (p *ModelAPIPlugin) chatToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	if !p.enabled {
		return &plugin.ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	// 解析提供商
	providerStr, _ := args["provider"].(string)
	if providerStr == "" {
		providerStr = string(p.config.DefaultProvider)
	}

	provider := ModelProvider(providerStr)

	// 检查提供商配置
	config, exists := p.config.Providers[providerStr]
	if !exists {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("不支持的提供商: %s", providerStr),
		}, nil
	}

	// 检查API密钥
	if config.APIKey == "" {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("请先配置 %s 的API密钥", providerStr),
		}, nil
	}

	// 解析消息
	messagesRaw, ok := args["messages"].([]interface{})
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "参数错误: messages必须是数组",
		}, nil
	}

	// 转换为消息格式
	messages := make([]Message, 0, len(messagesRaw))
	for _, msgRaw := range messagesRaw {
		msgMap, ok := msgRaw.(map[string]interface{})
		if !ok {
			return &plugin.ToolResult{
				Success: false,
				Error:   "参数错误: 消息格式不正确",
			}, nil
		}

		role, _ := msgMap["role"].(string)
		content, _ := msgMap["content"].(string)

		messages = append(messages, Message{
			Role:    role,
			Content: content,
		})
	}

	// 解析模型
	model, _ := args["model"].(string)
	if model == "" {
		model = config.Model
	}

	// 这里应该调用实际的模型API
	// 在实际实现中，会使用以下请求调用真正的API：
	// temperature := 0.7
	// if temp, ok := args["temperature"].(float64); ok {
	//     temperature = temp
	// }
	//
	// maxTokens := 2048
	// if tokens, ok := args["max_tokens"].(float64); ok {
	//     maxTokens = int(tokens)
	// }
	//
	// req := &ChatRequest{
	//     Model:       model,
	//     Messages:    messages,
	//     Temperature: temperature,
	//     MaxTokens:   maxTokens,
	//     Provider:    providerStr,
	// }
	// response, err := p.clients[provider].Chat(ctx, req)

	// 由于是示例，我们返回模拟响应
	response := &ChatResponse{
		ID:       fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Model:    model,
		Content:  fmt.Sprintf("这是 %s 提供商的模拟响应。在实际实现中，这里会调用真正的AI模型API。", providerStr),
		Provider: provider,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	// 发送事件通知
	p.EmitEvent(ctx, &plugin.Event{
		Type:      "model.chat.completed",
		Name:      "chat_completed",
		Data:      map[string]interface{}{"provider": providerStr, "model": model, "tokens": response.Usage.TotalTokens},
		Timestamp: time.Now(),
		Source:    "model_api_plugin",
	})

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"content":   response.Content,
			"model":     response.Model,
			"provider":  response.Provider,
			"usage":     response.Usage,
			"id":        response.ID,
			"timestamp": time.Now().Unix(),
		},
	}, nil
}

// configureToolHandler configure工具处理器
func (p *ModelAPIPlugin) configureToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	providerStr, ok := args["provider"].(string)
	if !ok || providerStr == "" {
		return &plugin.ToolResult{
			Success: false,
			Error:   "必须指定provider参数",
		}, nil
	}

	// 检查提供商是否存在
	config, exists := p.config.Providers[providerStr]
	if !exists {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("不支持的提供商: %s", providerStr),
		}, nil
	}

	updated := false

	// 更新api_key
	if apiKey, ok := args["api_key"].(string); ok && apiKey != "" {
		config.APIKey = apiKey
		updated = true
	}

	// 更新model
	if model, ok := args["model"].(string); ok && model != "" {
		config.Model = model
		updated = true
	}

	// 更新base_url
	if baseURL, ok := args["base_url"].(string); ok && baseURL != "" {
		config.BaseURL = baseURL
		updated = true
	}

	// 更新timeout
	if timeout, ok := args["timeout"].(float64); ok && timeout > 0 {
		config.Timeout = int(timeout)
		updated = true
	}

	if updated {
		// 保存配置
		p.config.Providers[providerStr] = config

		// 发送配置更新事件
		p.EmitEvent(ctx, &plugin.Event{
			Type:      "model.config.updated",
			Name:      "config_updated",
			Data:      map[string]interface{}{"provider": providerStr, "config": config},
			Timestamp: time.Now(),
			Source:    "model_api_plugin",
		})

		return &plugin.ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"message":  "配置更新成功",
				"provider": providerStr,
				"config":   config,
			},
		}, nil
	}

	return &plugin.ToolResult{
		Success: false,
		Error:   "没有提供有效的配置参数",
	}, nil
}

// statusToolHandler status工具处理器
func (p *ModelAPIPlugin) statusToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	status := "disabled"
	if p.enabled {
		status = "enabled"
	}

	// 统计配置情况
	configuredProviders := 0
	totalProviders := len(p.config.Providers)

	for _, config := range p.config.Providers {
		if config.APIKey != "" {
			configuredProviders++
		}
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"status":               status,
			"default_provider":     p.config.DefaultProvider,
			"total_providers":      totalProviders,
			"configured_providers": configuredProviders,
			"tools_count":          len(p.tools),
			"enabled":              p.enabled,
		},
	}, nil
}

// providersToolHandler providers工具处理器
func (p *ModelAPIPlugin) providersToolHandler(ctx context.Context, args map[string]interface{}) (*plugin.ToolResult, error) {
	providers := make([]map[string]interface{}, 0, len(p.config.Providers))

	for name, config := range p.config.Providers {
		providerInfo := map[string]interface{}{
			"name":        name,
			"base_url":    config.BaseURL,
			"model":       config.Model,
			"timeout":     config.Timeout,
			"api_key_set": config.APIKey != "",
		}
		providers = append(providers, providerInfo)
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		},
	}, nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// initClients 初始化客户端
func (p *ModelAPIPlugin) initClients() {
	// 这里可以初始化各个提供商的客户端
	// 在实际实现中，会为每个提供商创建具体的客户端实例
	if p.output != nil {
		p.output.Info("初始化模型客户端...")
	} else {
		fmt.Println("初始化模型客户端...")
	}
}

// subscribeEvents 订阅事件
func (p *ModelAPIPlugin) subscribeEvents() {
	// 订阅系统事件
	p.SubscribeEvent("system.startup", func(event *plugin.Event) error {
		if p.output != nil {
			p.output.Info("模型API插件收到系统启动事件")
		} else {
			fmt.Println("模型API插件收到系统启动事件")
		}
		return nil
	})

	p.SubscribeEvent("system.shutdown", func(event *plugin.Event) error {
		if p.output != nil {
			p.output.Info("模型API插件收到系统关闭事件")
		} else {
			fmt.Println("模型API插件收到系统关闭事件")
		}
		return nil
	})

	p.SubscribeEvent("model.api.error", func(event *plugin.Event) error {
		if p.output != nil {
			p.output.Printf("模型API插件收到错误事件: %v", event.Data)
		} else {
			fmt.Printf("模型API插件收到错误事件: %v\n", event.Data)
		}
		return nil
	})
}

// ============================================================================
// 插件导出函数
// ============================================================================

// NewPlugin 创建插件实例（供插件管理器调用）
func NewPlugin() plugin.Plugin {
	return NewModelAPIPlugin()
}

// GetPluginInfo 获取插件信息（供Yaegi调用）
func GetPluginInfo() *plugin.PluginInfo {
	return &PluginInfo
}
