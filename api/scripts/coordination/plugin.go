// 智慧体管理插件 - Yaegi 脚本插件
// 协调多智慧体，管理智慧体生命周期和交互

package coordination

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
	Name:         "智慧体管理插件",
	Version:      "1.0.0",
	Description:  "协调多智慧体，管理智慧体生命周期和交互",
	Author:       "OpenClaw System",
	Dependencies: []string{"context", "model"},
	Metadata: map[string]string{
		"category": "system",
		"type":     "coordination",
		"tags":     "agent,coordination,management,multi-agent",
	},
}

// ============================================================================
// 数据类型定义
// ============================================================================

// AgentType 智慧体类型
type AgentType string

const (
	AgentTypePrimary   AgentType = "primary"   // 主智慧体（猫猫）
	AgentTypeSpecial   AgentType = "special"   // 特殊智慧体
	AgentTypeGeneral   AgentType = "general"   // 通用智慧体
	AgentTypeAssistant AgentType = "assistant" // 助手智慧体
)

// AgentState 智慧体状态
type AgentState string

const (
	AgentStateActive     AgentState = "active"     // 活跃
	AgentStateInactive   AgentState = "inactive"   // 非活跃
	AgentStateSleeping   AgentState = "sleeping"   // 休眠
	AgentStateBusy       AgentState = "busy"       // 忙碌
	AgentStateError      AgentState = "error"      // 错误
	AgentStateTerminated AgentState = "terminated" // 终止
)

// AgentCapability 智慧体能力
type AgentCapability string

const (
	CapabilityReasoning     AgentCapability = "reasoning"     // 推理能力
	CapabilityPlanning      AgentCapability = "planning"      // 规划能力
	CapabilityExecution     AgentCapability = "execution"     // 执行能力
	CapabilityMemory        AgentCapability = "memory"        // 记忆能力
	CapabilityLearning      AgentCapability = "learning"      // 学习能力
	CapabilityToolUse       AgentCapability = "tool_use"      // 工具使用
	CapabilityCommunication AgentCapability = "communication" // 通信能力
)

// AgentConfig 智慧体配置
type AgentConfig struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         AgentType              `json:"type"`
	Capabilities []AgentCapability      `json:"capabilities"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	MemoryLimit  int                    `json:"memory_limit"`
	Tools        []string               `json:"tools"`
	Routes       []string               `json:"routes"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// Agent 智慧体定义
type Agent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         AgentType              `json:"type"`
	State        AgentState             `json:"state"`
	Capabilities []AgentCapability      `json:"capabilities"`
	Config       AgentConfig            `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActive   time.Time              `json:"last_active"`
	ParentID     string                 `json:"parent_id,omitempty"`
	Children     []string               `json:"children,omitempty"`
	ContextID    string                 `json:"context_id"`
	Stats        AgentStats             `json:"stats"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AgentStats 智慧体统计信息
type AgentStats struct {
	TotalInteractions int           `json:"total_interactions"`
	SuccessfulCalls   int           `json:"successful_calls"`
	FailedCalls       int           `json:"failed_calls"`
	TotalTokensUsed   int           `json:"total_tokens_used"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	LastCallTime      time.Time     `json:"last_call_time"`
	CreatedAt         time.Time     `json:"created_at"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// CoordinationRequest 协调请求
type CoordinationRequest struct {
	SourceAgentID string                 `json:"source_agent_id"`
	TargetAgentID string                 `json:"target_agent_id"`
	Task          string                 `json:"task"`
	Parameters    map[string]interface{} `json:"parameters"`
	Priority      int                    `json:"priority"`
	Timeout       time.Duration          `json:"timeout"`
}

// CoordinationResponse 协调响应
type CoordinationResponse struct {
	RequestID     string        `json:"request_id"`
	Success       bool          `json:"success"`
	Result        interface{}   `json:"result,omitempty"`
	Error         string        `json:"error,omitempty"`
	ExecutedBy    string        `json:"executed_by"`
	ExecutionTime time.Duration `json:"execution_time"`
	Timestamp     time.Time     `json:"timestamp"`
}

// ============================================================================
// 智慧体管理器
// ============================================================================

// AgentManager 智慧体管理器
type AgentManager struct {
	agents      map[string]*Agent
	mu          sync.RWMutex
	coordinator *Coordinator
	eventBus    *EventBus
}

// NewAgentManager 创建新的智慧体管理器
func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents:      make(map[string]*Agent),
		coordinator: NewCoordinator(),
		eventBus:    NewEventBus(),
	}
}

// CreateAgent 创建智慧体
func (m *AgentManager) CreateAgent(config AgentConfig) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一ID
	agentID := generateAgentID(config.Name)

	// 检查是否已存在
	if _, exists := m.agents[agentID]; exists {
		return nil, fmt.Errorf("智慧体已存在: %s", agentID)
	}

	// 创建智慧体
	agent := &Agent{
		ID:           agentID,
		Name:         config.Name,
		Description:  config.Description,
		Type:         config.Type,
		State:        AgentStateActive,
		Capabilities: config.Capabilities,
		Config:       config,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		ContextID:    "", // 将在上下文管理器中创建
		Stats: AgentStats{
			TotalInteractions: 0,
			SuccessfulCalls:   0,
			FailedCalls:       0,
			TotalTokensUsed:   0,
			AvgResponseTime:   0,
			LastCallTime:      time.Time{},
			CreatedAt:         time.Now(),
			LastUpdated:       time.Now(),
		},
		Metadata: make(map[string]interface{}),
	}

	// 添加到管理器
	m.agents[agentID] = agent

	// 发布事件
	m.eventBus.Publish("agent.created", map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agent.Name,
		"agent_type": string(agent.Type),
		"timestamp":  time.Now().Unix(),
	})

	return agent, nil
}

// GetAgent 获取智慧体
func (m *AgentManager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	agent, exists := m.agents[agentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("智慧体不存在: %s", agentID)
	}

	return agent, nil
}

// UpdateAgent 更新智慧体
func (m *AgentManager) UpdateAgent(agentID string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("智慧体不存在: %s", agentID)
	}

	// 应用更新
	for key, value := range updates {
		switch key {
		case "state":
			if state, ok := value.(string); ok {
				agent.State = AgentState(state)
			}
		case "config":
			if config, ok := value.(AgentConfig); ok {
				agent.Config = config
			}
		case "metadata":
			if metadata, ok := value.(map[string]interface{}); ok {
				agent.Metadata = metadata
			}
		case "stats":
			if stats, ok := value.(AgentStats); ok {
				agent.Stats = stats
			}
		}
	}

	agent.Stats.LastUpdated = time.Now()

	// 发布事件
	m.eventBus.Publish("agent.updated", map[string]interface{}{
		"agent_id":  agentID,
		"updates":   updates,
		"timestamp": time.Now().Unix(),
	})

	return nil
}

// DeleteAgent 删除智慧体
func (m *AgentManager) DeleteAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("智慧体不存在: %s", agentID)
	}

	// 从管理器中删除
	delete(m.agents, agentID)

	// 发布事件
	m.eventBus.Publish("agent.deleted", map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agent.Name,
		"timestamp":  time.Now().Unix(),
	})

	return nil
}

// ListAgents 列出所有智慧体
func (m *AgentManager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}

	return agents
}

// GetAgentStats 获取智慧体统计信息
func (m *AgentManager) GetAgentStats(agentID string) (*AgentStats, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		return nil, err
	}

	return &agent.Stats, nil
}

// UpdateAgentStats 更新智慧体统计信息
func (m *AgentManager) UpdateAgentStats(agentID string, success bool, tokensUsed int, responseTime time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("智慧体不存在: %s", agentID)
	}

	agent.Stats.TotalInteractions++
	if success {
		agent.Stats.SuccessfulCalls++
	} else {
		agent.Stats.FailedCalls++
	}
	agent.Stats.TotalTokensUsed += tokensUsed

	// 更新平均响应时间
	if agent.Stats.TotalInteractions > 0 {
		totalTime := agent.Stats.AvgResponseTime * time.Duration(agent.Stats.TotalInteractions-1)
		agent.Stats.AvgResponseTime = (totalTime + responseTime) / time.Duration(agent.Stats.TotalInteractions)
	}

	agent.Stats.LastCallTime = time.Now()
	agent.Stats.LastUpdated = time.Now()
	agent.LastActive = time.Now()

	return nil
}

// CoordinateRequest 协调请求
func (m *AgentManager) CoordinateRequest(request CoordinationRequest) (*CoordinationResponse, error) {
	return m.coordinator.Coordinate(request)
}

// ============================================================================
// 协调器
// ============================================================================

// Coordinator 协调器
type Coordinator struct {
	requests map[string]*CoordinationRequest
	mu       sync.RWMutex
}

// NewCoordinator 创建新的协调器
func NewCoordinator() *Coordinator {
	return &Coordinator{
		requests: make(map[string]*CoordinationRequest),
	}
}

// Coordinate 协调请求
func (c *Coordinator) Coordinate(request CoordinationRequest) (*CoordinationResponse, error) {
	requestID := generateRequestID()

	c.mu.Lock()
	c.requests[requestID] = &request
	c.mu.Unlock()

	// 这里应该实现实际的协调逻辑
	// 包括：任务分配、优先级处理、超时控制等

	// 模拟协调过程
	startTime := time.Now()
	time.Sleep(100 * time.Millisecond) // 模拟处理时间
	executionTime := time.Since(startTime)

	response := &CoordinationResponse{
		RequestID:     requestID,
		Success:       true,
		Result:        map[string]interface{}{"message": "请求已协调", "task": request.Task},
		ExecutedBy:    request.TargetAgentID,
		ExecutionTime: executionTime,
		Timestamp:     time.Now(),
	}

	// 清理请求记录
	c.mu.Lock()
	delete(c.requests, requestID)
	c.mu.Unlock()

	return response, nil
}

// ============================================================================
// 事件总线
// ============================================================================

// EventBus 简单事件总线
type EventBus struct {
	handlers map[string][]func(data map[string]interface{})
	mu       sync.RWMutex
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]func(data map[string]interface{})),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(event string, handler func(data map[string]interface{})) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[event] = append(eb.handlers[event], handler)
}

// Publish 发布事件
func (eb *EventBus) Publish(event string, data map[string]interface{}) {
	eb.mu.RLock()
	handlers := eb.handlers[event]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// ============================================================================
// 插件实现
// ============================================================================

// CoordinationPlugin 智慧体管理插件
type CoordinationPlugin struct {
	*plugin.BasePlugin
	manager *AgentManager
}

// NewPlugin 必须导出，供插件管理器调用
func NewPlugin() plugin.Plugin {
	return &CoordinationPlugin{
		BasePlugin: plugin.NewBasePlugin(PluginInfo, nil),
	}
}

// Init 初始化插件
func (p *CoordinationPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 初始化智慧体管理器
	p.manager = NewAgentManager()

	// 注册工具
	if err := p.registerTools(); err != nil {
		return fmt.Errorf("注册工具失败: %w", err)
	}

	// 订阅事件
	if err := p.subscribeEvents(); err != nil {
		return fmt.Errorf("订阅事件失败: %w", err)
	}

	// 创建默认主智慧体（猫猫）
	if err := p.createDefaultPrimaryAgent(); err != nil {
		fmt.Printf("⚠️ 创建默认主智慧体失败: %v\n", err)
	}

	fmt.Println("✅ 智慧体管理插件初始化完成")
	return nil
}

// Shutdown 关闭插件
func (p *CoordinationPlugin) Shutdown(ctx context.Context) error {
	fmt.Println("🔄 智慧体管理插件关闭...")

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// ============================================================================
// 工具注册
// ============================================================================

// registerTools 注册插件提供的工具
func (p *CoordinationPlugin) registerTools() error {
	// 注册创建智慧体工具
	createTool := plugin.ToolDefinition{
		Name:        "agent.create",
		Description: "创建新的智慧体",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "智慧体名称",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "智慧体描述",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "智慧体类型 (primary, special, general, assistant)",
					"enum":        []string{"primary", "special", "general", "assistant"},
				},
				"capabilities": map[string]interface{}{
					"type":        "array",
					"description": "智慧体能力列表",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"reasoning", "planning", "execution", "memory", "learning", "tool_use", "communication"},
					},
				},
				"model": map[string]interface{}{
					"type":        "string",
					"description": "使用的模型",
				},
				"provider": map[string]interface{}{
					"type":        "string",
					"description": "模型提供商",
				},
				"temperature": map[string]interface{}{
					"type":        "number",
					"description": "温度参数",
					"minimum":     0.0,
					"maximum":     2.0,
				},
				"max_tokens": map[string]interface{}{
					"type":        "integer",
					"description": "最大token数",
					"minimum":     1,
					"maximum":     4096,
				},
				"memory_limit": map[string]interface{}{
					"type":        "integer",
					"description": "记忆限制",
					"minimum":     1,
				},
				"tools": map[string]interface{}{
					"type":        "array",
					"description": "可用工具列表",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"routes": map[string]interface{}{
					"type":        "array",
					"description": "可用路由列表",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"parameters": map[string]interface{}{
					"type":        "object",
					"description": "额外参数",
				},
			},
			Required: []string{"name", "description", "type"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(createTool, p.handleCreateAgent); err != nil {
		return fmt.Errorf("注册工具 agent.create 失败: %w", err)
	}

	// 注册获取智慧体工具
	getTool := plugin.ToolDefinition{
		Name:        "agent.get",
		Description: "获取智慧体信息",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
			},
			Required: []string{"agent_id"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(getTool, p.handleGetAgent); err != nil {
		return fmt.Errorf("注册工具 agent.get 失败: %w", err)
	}

	// 注册更新智慧体工具
	updateTool := plugin.ToolDefinition{
		Name:        "agent.update",
		Description: "更新智慧体信息",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
				"updates": map[string]interface{}{
					"type":        "object",
					"description": "要更新的字段",
				},
			},
			Required: []string{"agent_id", "updates"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(updateTool, p.handleUpdateAgent); err != nil {
		return fmt.Errorf("注册工具 agent.update 失败: %w", err)
	}

	// 注册删除智慧体工具
	deleteTool := plugin.ToolDefinition{
		Name:        "agent.delete",
		Description: "删除智慧体",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
			},
			Required: []string{"agent_id"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(deleteTool, p.handleDeleteAgent); err != nil {
		return fmt.Errorf("注册工具 agent.delete 失败: %w", err)
	}

	// 注册列出智慧体工具
	listTool := plugin.ToolDefinition{
		Name:        "agent.list",
		Description: "列出所有智慧体",
		InputSchema: &plugin.ToolSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(listTool, p.handleListAgents); err != nil {
		return fmt.Errorf("注册工具 agent.list 失败: %w", err)
	}

	// 注册协调请求工具
	coordinateTool := plugin.ToolDefinition{
		Name:        "agent.coordinate",
		Description: "协调智慧体间请求",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"source_agent_id": map[string]interface{}{
					"type":        "string",
					"description": "源智慧体ID",
				},
				"target_agent_id": map[string]interface{}{
					"type":        "string",
					"description": "目标智慧体ID",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "任务描述",
				},
				"parameters": map[string]interface{}{
					"type":        "object",
					"description": "任务参数",
				},
				"priority": map[string]interface{}{
					"type":        "integer",
					"description": "优先级 (1-10)",
					"minimum":     1,
					"maximum":     10,
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "超时时间（秒）",
					"minimum":     1,
				},
			},
			Required: []string{"source_agent_id", "target_agent_id", "task"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(coordinateTool, p.handleCoordinateRequest); err != nil {
		return fmt.Errorf("注册工具 agent.coordinate 失败: %w", err)
	}

	// 注册获取统计信息工具
	statsTool := plugin.ToolDefinition{
		Name:        "agent.stats",
		Description: "获取智慧体统计信息",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
			},
			Required: []string{"agent_id"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(statsTool, p.handleGetAgentStats); err != nil {
		return fmt.Errorf("注册工具 agent.stats 失败: %w", err)
	}

	fmt.Println("✅ 注册了 7 个智慧体管理工具")
	return nil
}

// ============================================================================
// 工具处理函数
// ============================================================================

// handleCreateAgent 处理创建智慧体请求
func (p *CoordinationPlugin) handleCreateAgent(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	name, ok := params["name"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "name 必须是字符串",
		}, nil
	}

	description, ok := params["description"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "description 必须是字符串",
		}, nil
	}

	typeStr, ok := params["type"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "type 必须是字符串",
		}, nil
	}

	// 解析能力列表
	var capabilities []AgentCapability
	if caps, ok := params["capabilities"].([]interface{}); ok {
		for _, cap := range caps {
			if capStr, ok := cap.(string); ok {
				capabilities = append(capabilities, AgentCapability(capStr))
			}
		}
	}

	// 解析配置
	config := AgentConfig{
		Name:         name,
		Description:  description,
		Type:         AgentType(typeStr),
		Capabilities: capabilities,
		Model:        "deepseek-ai/DeepSeek-V3",
		Provider:     "siliconflow",
		Temperature:  0.7,
		MaxTokens:    2048,
		MemoryLimit:  1000,
		Tools:        []string{},
		Routes:       []string{},
		Parameters:   make(map[string]interface{}),
	}

	// 覆盖默认配置
	if model, ok := params["model"].(string); ok && model != "" {
		config.Model = model
	}
	if provider, ok := params["provider"].(string); ok && provider != "" {
		config.Provider = provider
	}
	if temp, ok := params["temperature"].(float64); ok {
		config.Temperature = temp
	}
	if maxTokens, ok := params["max_tokens"].(float64); ok {
		config.MaxTokens = int(maxTokens)
	}
	if memoryLimit, ok := params["memory_limit"].(float64); ok {
		config.MemoryLimit = int(memoryLimit)
	}
	if tools, ok := params["tools"].([]interface{}); ok {
		for _, tool := range tools {
			if toolStr, ok := tool.(string); ok {
				config.Tools = append(config.Tools, toolStr)
			}
		}
	}
	if routes, ok := params["routes"].([]interface{}); ok {
		for _, route := range routes {
			if routeStr, ok := route.(string); ok {
				config.Routes = append(config.Routes, routeStr)
			}
		}
	}
	if parameters, ok := params["parameters"].(map[string]interface{}); ok {
		config.Parameters = parameters
	}

	// 创建智慧体
	agent, err := p.manager.CreateAgent(config)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("创建智慧体失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"agent_id":    agent.ID,
			"name":        agent.Name,
			"description": agent.Description,
			"type":        string(agent.Type),
			"state":       string(agent.State),
			"created_at":  agent.CreatedAt.Format(time.RFC3339),
			"context_id":  agent.ContextID,
		},
	}, nil
}

// handleGetAgent 处理获取智慧体请求
func (p *CoordinationPlugin) handleGetAgent(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	agent, err := p.manager.GetAgent(agentID)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("获取智慧体失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"id":           agent.ID,
			"name":         agent.Name,
			"description":  agent.Description,
			"type":         string(agent.Type),
			"state":        string(agent.State),
			"capabilities": agent.Capabilities,
			"created_at":   agent.CreatedAt.Format(time.RFC3339),
			"last_active":  agent.LastActive.Format(time.RFC3339),
			"parent_id":    agent.ParentID,
			"children":     agent.Children,
			"context_id":   agent.ContextID,
			"config":       agent.Config,
			"stats":        agent.Stats,
			"metadata":     agent.Metadata,
		},
	}, nil
}

// handleUpdateAgent 处理更新智慧体请求
func (p *CoordinationPlugin) handleUpdateAgent(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	updates, ok := params["updates"].(map[string]interface{})
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "updates 必须是对象",
		}, nil
	}

	if err := p.manager.UpdateAgent(agentID, updates); err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("更新智慧体失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "智慧体更新成功",
		},
	}, nil
}

// handleDeleteAgent 处理删除智慧体请求
func (p *CoordinationPlugin) handleDeleteAgent(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	if err := p.manager.DeleteAgent(agentID); err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("删除智慧体失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "智慧体删除成功",
		},
	}, nil
}

// handleListAgents 处理列出智慧体请求
func (p *CoordinationPlugin) handleListAgents(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agents := p.manager.ListAgents()

	result := make([]map[string]interface{}, 0, len(agents))
	for _, agent := range agents {
		result = append(result, map[string]interface{}{
			"id":           agent.ID,
			"name":         agent.Name,
			"description":  agent.Description,
			"type":         string(agent.Type),
			"state":        string(agent.State),
			"created_at":   agent.CreatedAt.Format(time.RFC3339),
			"last_active":  agent.LastActive.Format(time.RFC3339),
			"capabilities": len(agent.Capabilities),
			"tools":        len(agent.Config.Tools),
			"routes":       len(agent.Config.Routes),
			"total_calls":  agent.Stats.TotalInteractions,
			"success_rate": calculateSuccessRate(agent.Stats),
		})
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"agents": result,
			"count":  len(result),
		},
	}, nil
}

// handleCoordinateRequest 处理协调请求
func (p *CoordinationPlugin) handleCoordinateRequest(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	sourceAgentID, ok := params["source_agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "source_agent_id 必须是字符串",
		}, nil
	}

	targetAgentID, ok := params["target_agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "target_agent_id 必须是字符串",
		}, nil
	}

	task, ok := params["task"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "task 必须是字符串",
		}, nil
	}

	request := CoordinationRequest{
		SourceAgentID: sourceAgentID,
		TargetAgentID: targetAgentID,
		Task:          task,
		Parameters:    make(map[string]interface{}),
		Priority:      5,
		Timeout:       30 * time.Second,
	}

	// 解析参数
	if parameters, ok := params["parameters"].(map[string]interface{}); ok {
		request.Parameters = parameters
	}
	if priority, ok := params["priority"].(float64); ok {
		request.Priority = int(priority)
	}
	if timeout, ok := params["timeout"].(float64); ok {
		request.Timeout = time.Duration(timeout) * time.Second
	}

	response, err := p.manager.CoordinateRequest(request)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("协调请求失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"request_id":     response.RequestID,
			"success":        response.Success,
			"result":         response.Result,
			"error":          response.Error,
			"executed_by":    response.ExecutedBy,
			"execution_time": response.ExecutionTime.String(),
			"timestamp":      response.Timestamp.Format(time.RFC3339),
		},
	}, nil
}

// handleGetAgentStats 处理获取智慧体统计信息请求
func (p *CoordinationPlugin) handleGetAgentStats(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	stats, err := p.manager.GetAgentStats(agentID)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("获取智慧体统计信息失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"total_interactions": stats.TotalInteractions,
			"successful_calls":   stats.SuccessfulCalls,
			"failed_calls":       stats.FailedCalls,
			"success_rate":       calculateSuccessRate(*stats),
			"total_tokens_used":  stats.TotalTokensUsed,
			"avg_response_time":  stats.AvgResponseTime.String(),
			"last_call_time":     stats.LastCallTime.Format(time.RFC3339),
			"created_at":         stats.CreatedAt.Format(time.RFC3339),
			"last_updated":       stats.LastUpdated.Format(time.RFC3339),
		},
	}, nil
}

// ============================================================================
// 辅助函数
// ============================================================================

// generateAgentID 生成智慧体ID
func generateAgentID(name string) string {
	return fmt.Sprintf("agent_%s_%d", sanitizeName(name), time.Now().UnixNano())
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), randomString(6))
}

// sanitizeName 清理名称
func sanitizeName(name string) string {
	// 移除特殊字符，只保留字母、数字和下划线
	result := ""
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			result += string(ch)
		} else if ch == ' ' {
			result += "_"
		}
	}
	if result == "" {
		result = "agent"
	}
	return result
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// calculateSuccessRate 计算成功率
func calculateSuccessRate(stats AgentStats) float64 {
	if stats.TotalInteractions == 0 {
		return 0.0
	}
	return float64(stats.SuccessfulCalls) / float64(stats.TotalInteractions) * 100
}

// createDefaultPrimaryAgent 创建默认主智慧体（猫猫）
func (p *CoordinationPlugin) createDefaultPrimaryAgent() error {
	config := AgentConfig{
		Name:        "猫猫",
		Description: "OpenClaw系统主智慧体，负责整体协调和决策",
		Type:        AgentTypePrimary,
		Capabilities: []AgentCapability{
			CapabilityReasoning,
			CapabilityPlanning,
			CapabilityExecution,
			CapabilityMemory,
			CapabilityLearning,
			CapabilityToolUse,
			CapabilityCommunication,
		},
		Model:       "deepseek-ai/DeepSeek-V3",
		Provider:    "siliconflow",
		Temperature: 0.7,
		MaxTokens:   4096,
		MemoryLimit: 5000,
		Tools:       []string{},
		Routes:      []string{},
		Parameters: map[string]interface{}{
			"role":           "primary_agent",
			"system_prompt":  "你是OpenClaw系统的主智慧体，负责协调和管理其他智慧体，处理复杂任务。",
			"max_context":    10000,
			"thinking_depth": "deep",
		},
	}

	agent, err := p.manager.CreateAgent(config)
	if err != nil {
		return err
	}

	fmt.Printf("✅ 创建默认主智慧体: %s (ID: %s)\n", agent.Name, agent.ID)
	return nil
}

// ============================================================================
// 事件订阅
// ============================================================================

// subscribeEvents 订阅相关事件
func (p *CoordinationPlugin) subscribeEvents() error {
	// 订阅上下文创建事件
	p.manager.eventBus.Subscribe("context.created", func(data map[string]interface{}) {
		agentID, ok := data["agent_id"].(string)
		if !ok {
			return
		}

		contextID, ok := data["context_id"].(string)
		if !ok {
			return
		}

		// 更新智慧体的上下文ID
		if err := p.manager.UpdateAgent(agentID, map[string]interface{}{
			"context_id": contextID,
		}); err != nil {
			fmt.Printf("⚠️ 更新智慧体 %s 上下文ID失败: %v\n", agentID, err)
		}
	})

	// 订阅工具注册事件
	p.manager.eventBus.Subscribe("tool.registered", func(data map[string]interface{}) {
		agentID, ok := data["agent_id"].(string)
		if !ok {
			return
		}

		toolName, ok := data["tool_name"].(string)
		if !ok {
			return
		}

		// 获取智慧体
		agent, err := p.manager.GetAgent(agentID)
		if err != nil {
			return
		}

		// 检查工具是否已存在
		for _, existingTool := range agent.Config.Tools {
			if existingTool == toolName {
				return
			}
		}

		// 添加工具到智慧体配置
		agent.Config.Tools = append(agent.Config.Tools, toolName)
		if err := p.manager.UpdateAgent(agentID, map[string]interface{}{
			"config": agent.Config,
		}); err != nil {
			fmt.Printf("⚠️ 更新智慧体 %s 工具列表失败: %v\n", agentID, err)
		}
	})

	// 订阅路由注册事件
	p.manager.eventBus.Subscribe("route.registered", func(data map[string]interface{}) {
		agentID, ok := data["agent_id"].(string)
		if !ok {
			return
		}

		routeName, ok := data["route_name"].(string)
		if !ok {
			return
		}

		// 获取智慧体
		agent, err := p.manager.GetAgent(agentID)
		if err != nil {
			return
		}

		// 检查路由是否已存在
		for _, existingRoute := range agent.Config.Routes {
			if existingRoute == routeName {
				return
			}
		}

		// 添加路由到智慧体配置
		agent.Config.Routes = append(agent.Config.Routes, routeName)
		if err := p.manager.UpdateAgent(agentID, map[string]interface{}{
			"config": agent.Config,
		}); err != nil {
			fmt.Printf("⚠️ 更新智慧体 %s 路由列表失败: %v\n", agentID, err)
		}
	})

	// 订阅模型调用事件
	p.manager.eventBus.Subscribe("model.chat.completed", func(data map[string]interface{}) {
		agentID, ok := data["agent_id"].(string)
		if !ok {
			return
		}

		success, _ := data["success"].(bool)
		tokensUsed, _ := data["tokens_used"].(float64)
		responseTime, _ := data["response_time"].(float64)

		// 更新智慧体统计信息
		if err := p.manager.UpdateAgentStats(agentID, success, int(tokensUsed), time.Duration(responseTime)*time.Millisecond); err != nil {
			fmt.Printf("⚠️ 更新智慧体 %s 统计信息失败: %v\n", agentID, err)
		}
	})

	fmt.Println("✅ 智慧体管理插件事件订阅完成")
	return nil
}
