// 工具管理插件 - Yaegi 脚本插件
// 作为中间层，让智慧体返回的工具调用对现有功能进行调用

package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"api/plugin"
)

// ============================================================================
// 插件信息定义
// ============================================================================

// PluginInfo 必须导出，供 Yaegi 加载器识别
var PluginInfo = &plugin.PluginInfo{
	Name:         "工具管理插件",
	Version:      "1.0.0",
	Description:  "工具管理中间层，协调工具调用和功能集成",
	Author:       "OpenClaw System",
	Dependencies: []string{"context", "coordination"},
	Metadata: map[string]string{
		"category": "system",
		"type":     "tools",
		"tags":     "tools,management,middleware,integration",
	},
}

// ============================================================================
// 数据类型定义
// ============================================================================

// ToolCategory 工具类别
type ToolCategory string

const (
	ToolCategoryMemory   ToolCategory = "memory"   // 记忆管理
	ToolCategoryFile     ToolCategory = "file"     // 文件管理
	ToolCategoryConfig   ToolCategory = "config"   // 参数管理
	ToolCategorySystem   ToolCategory = "system"   // 系统管理
	ToolCategoryNetwork  ToolCategory = "network"  // 网络管理
	ToolCategoryData     ToolCategory = "data"     // 数据处理
	ToolCategoryAnalysis ToolCategory = "analysis" // 分析工具
	ToolCategoryUtility  ToolCategory = "utility"  // 实用工具
)

// ToolPermission 工具权限
type ToolPermission string

const (
	PermissionRead    ToolPermission = "read"
	PermissionWrite   ToolPermission = "write"
	PermissionExecute ToolPermission = "execute"
	PermissionAdmin   ToolPermission = "admin"
)

// ToolDefinition 工具定义
type ToolDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     ToolCategory           `json:"category"`
	Permissions  []ToolPermission       `json:"permissions"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema"`
	Handler      string                 `json:"handler"` // 处理函数名称
	Enabled      bool                   `json:"enabled"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutionRequest 工具执行请求
type ToolExecutionRequest struct {
	ToolID    string                 `json:"tool_id"`
	AgentID   string                 `json:"agent_id"`
	Arguments map[string]interface{} `json:"arguments"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Priority  int                    `json:"priority"`
	Timeout   time.Duration          `json:"timeout"`
}

// ToolExecutionResponse 工具执行响应
type ToolExecutionResponse struct {
	RequestID   string                 `json:"request_id"`
	Success     bool                   `json:"success"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ExecutionID string                 `json:"execution_id"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]*ToolDefinition
	mu    sync.RWMutex
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDefinition),
	}
}

// RegisterTool 注册工具
func (r *ToolRegistry) RegisterTool(tool *ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.ID]; exists {
		return fmt.Errorf("工具已存在: %s", tool.ID)
	}

	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()
	r.tools[tool.ID] = tool

	return nil
}

// GetTool 获取工具
func (r *ToolRegistry) GetTool(toolID string) (*ToolDefinition, error) {
	r.mu.RLock()
	tool, exists := r.tools[toolID]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("工具不存在: %s", toolID)
	}

	return tool, nil
}

// UpdateTool 更新工具
func (r *ToolRegistry) UpdateTool(toolID string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tool, exists := r.tools[toolID]
	if !exists {
		return fmt.Errorf("工具不存在: %s", toolID)
	}

	// 应用更新
	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				tool.Name = name
			}
		case "description":
			if description, ok := value.(string); ok {
				tool.Description = description
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				tool.Enabled = enabled
			}
		case "metadata":
			if metadata, ok := value.(map[string]interface{}); ok {
				tool.Metadata = metadata
			}
		}
	}

	tool.UpdatedAt = time.Now()

	return nil
}

// UnregisterTool 注销工具
func (r *ToolRegistry) UnregisterTool(toolID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolID]; !exists {
		return fmt.Errorf("工具不存在: %s", toolID)
	}

	delete(r.tools, toolID)
	return nil
}

// ListTools 列出所有工具
func (r *ToolRegistry) ListTools() []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ListToolsByCategory 按类别列出工具
func (r *ToolRegistry) ListToolsByCategory(category ToolCategory) []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*ToolDefinition, 0)
	for _, tool := range r.tools {
		if tool.Category == category {
			tools = append(tools, tool)
		}
	}

	return tools
}

// ============================================================================
// 工具执行器
// ============================================================================

// ToolExecutor 工具执行器
type ToolExecutor struct {
	registry *ToolRegistry
	handlers map[string]func(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error)
	mu       sync.RWMutex
}

// NewToolExecutor 创建新的工具执行器
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		registry: NewToolRegistry(),
		handlers: make(map[string]func(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error)),
	}
}

// RegisterHandler 注册处理函数
func (e *ToolExecutor) RegisterHandler(toolID string, handler func(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers[toolID] = handler
}

// ExecuteTool 执行工具
func (e *ToolExecutor) ExecuteTool(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 获取工具定义
	tool, err := e.registry.GetTool(req.ToolID)
	if err != nil {
		return &ToolExecutionResponse{
			Success:   false,
			Error:     fmt.Sprintf("获取工具失败: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	// 检查工具是否启用
	if !tool.Enabled {
		return &ToolExecutionResponse{
			Success:   false,
			Error:     "工具已禁用",
			Timestamp: time.Now(),
		}, nil
	}

	// 获取处理函数
	e.mu.RLock()
	handler, exists := e.handlers[req.ToolID]
	e.mu.RUnlock()

	if !exists {
		return &ToolExecutionResponse{
			Success:   false,
			Error:     "工具处理函数未注册",
			Timestamp: time.Now(),
		}, nil
	}

	// 执行工具
	startTime := time.Now()
	response, err := handler(ctx, req)
	duration := time.Since(startTime)

	if response == nil {
		response = &ToolExecutionResponse{}
	}

	response.Duration = duration
	response.Timestamp = time.Now()

	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("工具执行失败: %v", err)
	}

	return response, nil
}

// ============================================================================
// 插件实现
// ============================================================================

// ToolsPlugin 工具管理插件
type ToolsPlugin struct {
	*plugin.BasePlugin
	registry *ToolRegistry
	executor *ToolExecutor
}

// NewPlugin 必须导出，供插件管理器调用
func NewPlugin() plugin.Plugin {
	return &ToolsPlugin{
		BasePlugin: plugin.NewBasePlugin(PluginInfo, nil),
	}
}

// Init 初始化插件
func (p *ToolsPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 初始化工具注册表和执行器
	p.registry = NewToolRegistry()
	p.executor = NewToolExecutor()

	// 注册默认工具
	if err := p.registerDefaultTools(); err != nil {
		return fmt.Errorf("注册默认工具失败: %w", err)
	}

	// 注册工具处理函数
	if err := p.registerToolHandlers(); err != nil {
		return fmt.Errorf("注册工具处理函数失败: %w", err)
	}

	// 注册插件工具
	if err := p.registerTools(); err != nil {
		return fmt.Errorf("注册插件工具失败: %w", err)
	}

	fmt.Println("✅ 工具管理插件初始化完成")
	return nil
}

// Shutdown 关闭插件
func (p *ToolsPlugin) Shutdown(ctx context.Context) error {
	fmt.Println("🔄 工具管理插件关闭...")

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// ============================================================================
// 默认工具注册
// ============================================================================

// registerDefaultTools 注册默认工具
func (p *ToolsPlugin) registerDefaultTools() error {
	// 记忆管理工具
	memoryTools := []*ToolDefinition{
		{
			ID:          "memory.store",
			Name:        "存储记忆",
			Description: "将信息存储到记忆系统中",
			Category:    ToolCategoryMemory,
			Permissions: []ToolPermission{PermissionWrite},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "记忆键",
					},
					"value": map[string]interface{}{
						"type":        "any",
						"description": "记忆值",
					},
					"ttl": map[string]interface{}{
						"type":        "integer",
						"description": "生存时间（秒）",
					},
				},
				"required": []string{"key", "value"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"message": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Handler: "handleMemoryStore",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
		{
			ID:          "memory.retrieve",
			Name:        "检索记忆",
			Description: "从记忆系统中检索信息",
			Category:    ToolCategoryMemory,
			Permissions: []ToolPermission{PermissionRead},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "记忆键",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "搜索模式（可选）",
					},
				},
				"required": []string{"key"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"value": map[string]interface{}{
						"type": "any",
					},
				},
			},
			Handler: "handleMemoryRetrieve",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
	}

	// 文件管理工具
	fileTools := []*ToolDefinition{
		{
			ID:          "file.read",
			Name:        "读取文件",
			Description: "读取文件内容",
			Category:    ToolCategoryFile,
			Permissions: []ToolPermission{PermissionRead},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
					"encoding": map[string]interface{}{
						"type":        "string",
						"description": "文件编码",
						"default":     "utf-8",
					},
				},
				"required": []string{"path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"content": map[string]interface{}{
						"type": "string",
					},
					"size": map[string]interface{}{
						"type": "integer",
					},
				},
			},
			Handler: "handleFileRead",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
		{
			ID:          "file.write",
			Name:        "写入文件",
			Description: "写入文件内容",
			Category:    ToolCategoryFile,
			Permissions: []ToolPermission{PermissionWrite},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "文件内容",
					},
					"encoding": map[string]interface{}{
						"type":        "string",
						"description": "文件编码",
						"default":     "utf-8",
					},
					"append": map[string]interface{}{
						"type":        "boolean",
						"description": "是否追加",
						"default":     false,
					},
				},
				"required": []string{"path", "content"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"message": map[string]interface{}{
						"type": "string",
					},
					"size": map[string]interface{}{
						"type": "integer",
					},
				},
			},
			Handler: "handleFileWrite",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
	}

	// 参数管理工具
	configTools := []*ToolDefinition{
		{
			ID:          "config.get",
			Name:        "获取配置",
			Description: "获取系统配置参数",
			Category:    ToolCategoryConfig,
			Permissions: []ToolPermission{PermissionRead},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "配置键",
					},
					"default": map[string]interface{}{
						"type":        "any",
						"description": "默认值",
					},
				},
				"required": []string{"key"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"value": map[string]interface{}{
						"type": "any",
					},
				},
			},
			Handler: "handleConfigGet",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
		{
			ID:          "config.set",
			Name:        "设置配置",
			Description: "设置系统配置参数",
			Category:    ToolCategoryConfig,
			Permissions: []ToolPermission{PermissionWrite},
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "配置键",
					},
					"value": map[string]interface{}{
						"type":        "any",
						"description": "配置值",
					},
				},
				"required": []string{"key", "value"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type": "boolean",
					},
					"message": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Handler: "handleConfigSet",
			Enabled: true,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"author":  "system",
			},
		},
	}

	// 注册所有工具
	allTools := append(memoryTools, fileTools...)
	allTools = append(allTools, configTools...)

	for _, tool := range allTools {
		if err := p.registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("注册工具 %s 失败: %w", tool.ID, err)
		}
	}

	fmt.Printf("✅ 注册了 %d 个默认工具\n", len(allTools))
	return nil
}

// ============================================================================
// 工具处理函数注册
// ============================================================================

// registerToolHandlers 注册工具处理函数
func (p *ToolsPlugin) registerToolHandlers() error {
	// 注册记忆存储处理函数
	p.executor.RegisterHandler("memory.store", p.handleMemoryStore)

	// 注册记忆检索处理函数
	p.executor.RegisterHandler("memory.retrieve", p.handleMemoryRetrieve)

	// 注册文件读取处理函数
	p.executor.RegisterHandler("file.read", p.handleFileRead)

	// 注册文件写入处理函数
	p.executor.RegisterHandler("file.write", p.handleFileWrite)

	// 注册配置获取处理函数
	p.executor.RegisterHandler("config.get", p.handleConfigGet)

	// 注册配置设置处理函数
	p.executor.RegisterHandler("config.set", p.handleConfigSet)

	fmt.Println("✅ 注册了 6 个工具处理函数")
	return nil
}

// ============================================================================
// 工具处理函数实现
// ============================================================================

// handleMemoryStore 处理记忆存储请求
func (p *ToolsPlugin) handleMemoryStore(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的记忆存储逻辑
	// 在实际系统中，会调用记忆管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"message": "记忆存储成功"},
		Timestamp: time.Now(),
	}, nil
}

// handleMemoryRetrieve 处理记忆检索请求
func (p *ToolsPlugin) handleMemoryRetrieve(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的记忆检索逻辑
	// 在实际系统中，会调用记忆管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"value": "检索到的记忆内容"},
		Timestamp: time.Now(),
	}, nil
}

// handleFileRead 处理文件读取请求
func (p *ToolsPlugin) handleFileRead(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的文件读取逻辑
	// 在实际系统中，会调用文件管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"content": "文件内容", "size": 1024},
		Timestamp: time.Now(),
	}, nil
}

// handleFileWrite 处理文件写入请求
func (p *ToolsPlugin) handleFileWrite(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的文件写入逻辑
	// 在实际系统中，会调用文件管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"message": "文件写入成功", "size": 1024},
		Timestamp: time.Now(),
	}, nil
}

// handleConfigGet 处理配置获取请求
func (p *ToolsPlugin) handleConfigGet(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的配置获取逻辑
	// 在实际系统中，会调用参数管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"value": "配置值"},
		Timestamp: time.Now(),
	}, nil
}

// handleConfigSet 处理配置设置请求
func (p *ToolsPlugin) handleConfigSet(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	// 这里应该实现实际的配置设置逻辑
	// 在实际系统中，会调用参数管理模块

	return &ToolExecutionResponse{
		Success:   true,
		Result:    map[string]interface{}{"message": "配置设置成功"},
		Timestamp: time.Now(),
	}, nil
}

// ============================================================================
// 插件工具注册
// ============================================================================

// registerTools 注册插件提供的工具
func (p *ToolsPlugin) registerTools() error {
	// 注册工具列表工具
	listTool := plugin.ToolDefinition{
		Name:        "tools.list",
		Description: "列出所有可用工具",
		InputSchema: &plugin.ToolSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(listTool, p.handleListTools); err != nil {
		return fmt.Errorf("注册工具 tools.list 失败: %w", err)
	}

	// 注册工具执行工具
	executeTool := plugin.ToolDefinition{
		Name:        "tools.execute",
		Description: "执行工具",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"tool_id": map[string]interface{}{
					"type":        "string",
					"description": "工具ID",
				},
				"arguments": map[string]interface{}{
					"type":        "object",
					"description": "工具参数",
				},
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
			},
			Required: []string{"tool_id", "arguments"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(executeTool, p.handleExecuteTool); err != nil {
		return fmt.Errorf("注册工具 tools.execute 失败: %w", err)
	}

	// 注册工具信息工具
	infoTool := plugin.ToolDefinition{
		Name:        "tools.info",
		Description: "获取工具详细信息",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"tool_id": map[string]interface{}{
					"type":        "string",
					"description": "工具ID",
				},
			},
			Required: []string{"tool_id"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(infoTool, p.handleToolInfo); err != nil {
		return fmt.Errorf("注册工具 tools.info 失败: %w", err)
	}

	fmt.Println("✅ 注册了 3 个插件工具")
	return nil
}

// ============================================================================
// 插件工具处理函数
// ============================================================================

// handleListTools 处理列出工具请求
func (p *ToolsPlugin) handleListTools(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	tools := p.registry.ListTools()

	result := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"id":          tool.ID,
			"name":        tool.Name,
			"description": tool.Description,
			"category":    string(tool.Category),
			"enabled":     tool.Enabled,
			"permissions": tool.Permissions,
		})
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"tools": result,
			"count": len(result),
		},
	}, nil
}

// handleExecuteTool 处理执行工具请求
func (p *ToolsPlugin) handleExecuteTool(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	toolID, ok := params["tool_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "tool_id 必须是字符串",
		}, nil
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "arguments 必须是对象",
		}, nil
	}

	agentID, _ := params["agent_id"].(string)

	req := &ToolExecutionRequest{
		ToolID:    toolID,
		AgentID:   agentID,
		Arguments: arguments,
		Priority:  5,
		Timeout:   30 * time.Second,
	}

	response, err := p.executor.ExecuteTool(ctx, req)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("执行工具失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: response.Success,
		Data:    response.Result,
		Error:   response.Error,
	}, nil
}

// handleToolInfo 处理获取工具信息请求
func (p *ToolsPlugin) handleToolInfo(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	toolID, ok := params["tool_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "tool_id 必须是字符串",
		}, nil
	}

	tool, err := p.registry.GetTool(toolID)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("获取工具信息失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"id":            tool.ID,
			"name":          tool.Name,
			"description":   tool.Description,
			"category":      string(tool.Category),
			"permissions":   tool.Permissions,
			"input_schema":  tool.InputSchema,
			"output_schema": tool.OutputSchema,
			"enabled":       tool.Enabled,
			"created_at":    tool.CreatedAt.Format(time.RFC3339),
			"updated_at":    tool.UpdatedAt.Format(time.RFC3339),
			"metadata":      tool.Metadata,
		},
	}, nil
}
