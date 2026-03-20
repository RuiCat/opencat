// 上下文管理插件 - Yaegi 脚本插件
// 管理多智慧体上下文，支持隔离和文件存储

package context

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"api/plugin"
)

// ============================================================================
// 插件信息定义
// ============================================================================

// PluginInfo 必须导出，供 Yaegi 加载器识别
var PluginInfo = &plugin.PluginInfo{
	Name:         "上下文管理插件",
	Version:      "1.0.0",
	Description:  "管理多智慧体上下文，支持隔离和文件存储",
	Author:       "OpenClaw System",
	Dependencies: []string{},
	Metadata: map[string]string{
		"category": "system",
		"type":     "context",
		"tags":     "context,agent,memory,storage",
	},
}

// ============================================================================
// 数据类型定义
// ============================================================================

// ContextState 上下文状态
type ContextState string

const (
	ContextStateActive   ContextState = "active"
	ContextStateInactive ContextState = "inactive"
	ContextStateArchived ContextState = "archived"
	ContextStateError    ContextState = "error"
)

// AgentContext 智慧体上下文
type AgentContext struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	ParentID     string                 `json:"parent_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	MemoryRefs   []string               `json:"memory_refs,omitempty"`
	Tools        []string               `json:"tools,omitempty"`
	Routes       []string               `json:"routes,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	State        ContextState           `json:"state"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// ContextStats 上下文统计信息
type ContextStats struct {
	TotalContexts   int       `json:"total_contexts"`
	ActiveContexts  int       `json:"active_contexts"`
	TotalMemoryRefs int       `json:"total_memory_refs"`
	TotalTools      int       `json:"total_tools"`
	TotalRoutes     int       `json:"total_routes"`
	CreatedAt       time.Time `json:"created_at"`
	LastUpdated     time.Time `json:"last_updated"`
	StorageSize     int64     `json:"storage_size"`
}

// ============================================================================
// 文件存储管理器
// ============================================================================

// FileContextManager 文件上下文管理器
type FileContextManager struct {
	baseDir  string
	contexts map[string]*AgentContext
	mu       sync.RWMutex
	eventBus *EventBus
}

// NewFileContextManager 创建新的文件上下文管理器
func NewFileContextManager() (*FileContextManager, error) {
	// 使用当前工作目录下的 data/contexts 目录
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取工作目录失败: %w", err)
	}

	baseDir := filepath.Join(cwd, "data", "contexts")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	manager := &FileContextManager{
		baseDir:  baseDir,
		contexts: make(map[string]*AgentContext),
		eventBus: NewEventBus(),
	}

	// 加载现有上下文
	if err := manager.loadAllContexts(); err != nil {
		fmt.Printf("加载现有上下文失败: %v\n", err)
		// 不返回错误，继续使用空管理器
	}

	return manager, nil
}

// loadAllContexts 加载所有上下文
func (m *FileContextManager) loadAllContexts() error {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，没有上下文
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(m.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("读取上下文文件失败 %s: %v\n", filePath, err)
			continue
		}

		var ctx AgentContext
		if err := json.Unmarshal(data, &ctx); err != nil {
			fmt.Printf("解析上下文文件失败 %s: %v\n", filePath, err)
			continue
		}

		m.contexts[ctx.AgentID] = &ctx
	}

	fmt.Printf("加载了 %d 个上下文\n", len(m.contexts))
	return nil
}

// saveContextToFile 保存上下文到文件
func (m *FileContextManager) saveContextToFile(ctx *AgentContext) error {
	filePath := filepath.Join(m.baseDir, ctx.AgentID+".json")

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化上下文失败: %w", err)
	}

	// 创建临时文件，然后原子重命名
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// ============================================================================
// 上下文管理方法
// ============================================================================

// CreateContext 创建新的上下文
func (m *FileContextManager) CreateContext(agentID, parentID string, config map[string]interface{}) (*AgentContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.contexts[agentID]; exists {
		return nil, fmt.Errorf("上下文已存在: %s", agentID)
	}

	// 创建上下文对象
	ctx := &AgentContext{
		ID:           generateID(),
		AgentID:      agentID,
		ParentID:     parentID,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		MemoryRefs:   []string{},
		Tools:        []string{},
		Routes:       []string{},
		Config:       config,
		State:        ContextStateActive,
		Data:         make(map[string]interface{}),
	}

	// 设置默认配置
	if ctx.Config == nil {
		ctx.Config = make(map[string]interface{})
	}
	if _, exists := ctx.Config["max_context_length"]; !exists {
		ctx.Config["max_context_length"] = 4096
	}
	if _, exists := ctx.Config["memory_limit"]; !exists {
		ctx.Config["memory_limit"] = 1000
	}

	// 保存到文件
	if err := m.saveContextToFile(ctx); err != nil {
		return nil, fmt.Errorf("保存上下文失败: %w", err)
	}

	// 更新内存缓存
	m.contexts[agentID] = ctx

	// 发布事件
	m.eventBus.Publish("context.created", map[string]interface{}{
		"agent_id":   agentID,
		"context_id": ctx.ID,
		"parent_id":  parentID,
		"timestamp":  time.Now().Unix(),
	})

	return ctx, nil
}

// GetContext 获取上下文
func (m *FileContextManager) GetContext(agentID string) (*AgentContext, error) {
	m.mu.RLock()
	ctx, exists := m.contexts[agentID]
	m.mu.RUnlock()

	if !exists {
		// 尝试从文件加载
		filePath := filepath.Join(m.baseDir, agentID+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("上下文不存在: %s", agentID)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("读取上下文文件失败: %w", err)
		}

		ctx = &AgentContext{}
		if err := json.Unmarshal(data, ctx); err != nil {
			return nil, fmt.Errorf("解析上下文文件失败: %w", err)
		}

		m.mu.Lock()
		m.contexts[agentID] = ctx
		m.mu.Unlock()
	}

	// 更新最后访问时间
	ctx.LastAccessed = time.Now()
	go m.saveContextToFile(ctx)

	return ctx, nil
}

// UpdateContext 更新上下文
func (m *FileContextManager) UpdateContext(agentID string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, exists := m.contexts[agentID]
	if !exists {
		return fmt.Errorf("上下文不存在: %s", agentID)
	}

	// 应用更新
	for key, value := range updates {
		switch key {
		case "config":
			if config, ok := value.(map[string]interface{}); ok {
				for k, v := range config {
					ctx.Config[k] = v
				}
			}
		case "data":
			if data, ok := value.(map[string]interface{}); ok {
				for k, v := range data {
					ctx.Data[k] = v
				}
			}
		case "state":
			if state, ok := value.(string); ok {
				ctx.State = ContextState(state)
			}
		case "tools":
			if tools, ok := value.([]string); ok {
				ctx.Tools = tools
			}
		case "routes":
			if routes, ok := value.([]string); ok {
				ctx.Routes = routes
			}
		case "memory_refs":
			if refs, ok := value.([]string); ok {
				ctx.MemoryRefs = refs
			}
		}
	}

	// 保存到文件
	if err := m.saveContextToFile(ctx); err != nil {
		return fmt.Errorf("保存上下文失败: %w", err)
	}

	// 发布事件
	m.eventBus.Publish("context.updated", map[string]interface{}{
		"agent_id":  agentID,
		"updates":   updates,
		"timestamp": time.Now().Unix(),
	})

	return nil
}

// DeleteContext 删除上下文
func (m *FileContextManager) DeleteContext(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	ctx, exists := m.contexts[agentID]
	if !exists {
		return fmt.Errorf("上下文不存在: %s", agentID)
	}

	// 删除文件
	filePath := filepath.Join(m.baseDir, agentID+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除上下文文件失败: %w", err)
	}

	// 从内存中删除
	delete(m.contexts, agentID)

	// 发布事件
	m.eventBus.Publish("context.deleted", map[string]interface{}{
		"agent_id":   agentID,
		"context_id": ctx.ID,
		"timestamp":  time.Now().Unix(),
	})

	return nil
}

// ListContexts 列出所有上下文
func (m *FileContextManager) ListContexts() []*AgentContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contexts := make([]*AgentContext, 0, len(m.contexts))
	for _, ctx := range m.contexts {
		contexts = append(contexts, ctx)
	}

	return contexts
}

// GetStats 获取统计信息
func (m *FileContextManager) GetStats() *ContextStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ContextStats{
		TotalContexts:   len(m.contexts),
		CreatedAt:       time.Now(),
		LastUpdated:     time.Now(),
		ActiveContexts:  0,
		TotalMemoryRefs: 0,
		TotalTools:      0,
		TotalRoutes:     0,
	}

	// 计算目录大小
	var totalSize int64
	entries, _ := os.ReadDir(m.baseDir)
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil {
			totalSize += info.Size()
		}
	}
	stats.StorageSize = totalSize

	// 计算其他统计
	for _, ctx := range m.contexts {
		if ctx.State == ContextStateActive {
			stats.ActiveContexts++
		}
		stats.TotalMemoryRefs += len(ctx.MemoryRefs)
		stats.TotalTools += len(ctx.Tools)
		stats.TotalRoutes += len(ctx.Routes)
	}

	return stats
}

// SaveAll 保存所有上下文
func (m *FileContextManager) SaveAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for _, ctx := range m.contexts {
		if err := m.saveContextToFile(ctx); err != nil {
			lastErr = err
			fmt.Printf("保存上下文 %s 失败: %v\n", ctx.AgentID, err)
		}
	}

	return lastErr
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
// 辅助函数
// ============================================================================

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("ctx_%d_%s", time.Now().UnixNano(), randomString(8))
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

// ============================================================================
// 插件实现
// ============================================================================

// ContextPlugin 上下文管理插件
type ContextPlugin struct {
	*plugin.BasePlugin
	manager *FileContextManager
}

// NewPlugin 必须导出，供插件管理器调用
func NewPlugin() plugin.Plugin {
	return &ContextPlugin{
		BasePlugin: plugin.NewBasePlugin(PluginInfo, nil),
	}
}

// Init 初始化插件
func (p *ContextPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 初始化文件上下文管理器
	manager, err := NewFileContextManager()
	if err != nil {
		return fmt.Errorf("初始化上下文管理器失败: %w", err)
	}

	p.manager = manager

	// 注册工具
	if err := p.registerTools(); err != nil {
		return fmt.Errorf("注册工具失败: %w", err)
	}

	// 订阅事件
	if err := p.subscribeEvents(); err != nil {
		return fmt.Errorf("订阅事件失败: %w", err)
	}

	fmt.Println("✅ 上下文管理插件初始化完成")
	return nil
}

// Shutdown 关闭插件
func (p *ContextPlugin) Shutdown(ctx context.Context) error {
	fmt.Println("🔄 上下文管理插件关闭...")

	// 保存所有上下文
	if p.manager != nil {
		if err := p.manager.SaveAll(); err != nil {
			fmt.Printf("⚠️ 保存上下文失败: %v\n", err)
		}
	}

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// ============================================================================
// 工具注册
// ============================================================================

// registerTools 注册插件提供的工具
func (p *ContextPlugin) registerTools() error {
	// 注册创建上下文工具
	createTool := plugin.ToolDefinition{
		Name:        "context.create",
		Description: "创建新的智慧体上下文",
		InputSchema: &plugin.ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "智慧体ID",
				},
				"parent_id": map[string]interface{}{
					"type":        "string",
					"description": "父智慧体ID（可选）",
				},
				"config": map[string]interface{}{
					"type":        "object",
					"description": "上下文配置（可选）",
				},
			},
			Required: []string{"agent_id"},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(createTool, p.handleCreateContext); err != nil {
		return fmt.Errorf("注册工具 context.create 失败: %w", err)
	}

	// 注册获取上下文工具
	getTool := plugin.ToolDefinition{
		Name:        "context.get",
		Description: "获取智慧体上下文",
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
	if err := p.RegisterTool(getTool, p.handleGetContext); err != nil {
		return fmt.Errorf("注册工具 context.get 失败: %w", err)
	}

	// 注册更新上下文工具
	updateTool := plugin.ToolDefinition{
		Name:        "context.update",
		Description: "更新智慧体上下文",
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
	if err := p.RegisterTool(updateTool, p.handleUpdateContext); err != nil {
		return fmt.Errorf("注册工具 context.update 失败: %w", err)
	}

	// 注册删除上下文工具
	deleteTool := plugin.ToolDefinition{
		Name:        "context.delete",
		Description: "删除智慧体上下文",
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
	if err := p.RegisterTool(deleteTool, p.handleDeleteContext); err != nil {
		return fmt.Errorf("注册工具 context.delete 失败: %w", err)
	}

	// 注册列出上下文工具
	listTool := plugin.ToolDefinition{
		Name:        "context.list",
		Description: "列出所有上下文",
		InputSchema: &plugin.ToolSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(listTool, p.handleListContexts); err != nil {
		return fmt.Errorf("注册工具 context.list 失败: %w", err)
	}

	// 注册获取统计信息工具
	statsTool := plugin.ToolDefinition{
		Name:        "context.stats",
		Description: "获取上下文统计信息",
		InputSchema: &plugin.ToolSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
		Enabled: true,
	}
	if err := p.RegisterTool(statsTool, p.handleGetStats); err != nil {
		return fmt.Errorf("注册工具 context.stats 失败: %w", err)
	}

	fmt.Println("✅ 注册了 6 个上下文管理工具")
	return nil
}

// ============================================================================
// 工具处理函数
// ============================================================================

// handleCreateContext 处理创建上下文请求
func (p *ContextPlugin) handleCreateContext(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	var parentID string
	if pid, ok := params["parent_id"].(string); ok {
		parentID = pid
	}

	var config map[string]interface{}
	if cfg, ok := params["config"].(map[string]interface{}); ok {
		config = cfg
	}

	context, err := p.manager.CreateContext(agentID, parentID, config)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("创建上下文失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"context_id": context.ID,
			"agent_id":   context.AgentID,
			"created_at": context.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// handleGetContext 处理获取上下文请求
func (p *ContextPlugin) handleGetContext(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	context, err := p.manager.GetContext(agentID)
	if err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("获取上下文失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"id":            context.ID,
			"agent_id":      context.AgentID,
			"parent_id":     context.ParentID,
			"created_at":    context.CreatedAt.Format(time.RFC3339),
			"last_accessed": context.LastAccessed.Format(time.RFC3339),
			"memory_refs":   context.MemoryRefs,
			"tools":         context.Tools,
			"routes":        context.Routes,
			"config":        context.Config,
			"state":         string(context.State),
			"data":          context.Data,
		},
	}, nil
}

// handleUpdateContext 处理更新上下文请求
func (p *ContextPlugin) handleUpdateContext(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
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

	if err := p.manager.UpdateContext(agentID, updates); err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("更新上下文失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "上下文更新成功",
		},
	}, nil
}

// handleDeleteContext 处理删除上下文请求
func (p *ContextPlugin) handleDeleteContext(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok {
		return &plugin.ToolResult{
			Success: false,
			Error:   "agent_id 必须是字符串",
		}, nil
	}

	if err := p.manager.DeleteContext(agentID); err != nil {
		return &plugin.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("删除上下文失败: %v", err),
		}, nil
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "上下文删除成功",
		},
	}, nil
}

// handleListContexts 处理列出上下文请求
func (p *ContextPlugin) handleListContexts(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	contexts := p.manager.ListContexts()

	result := make([]map[string]interface{}, 0, len(contexts))
	for _, ctx := range contexts {
		result = append(result, map[string]interface{}{
			"id":            ctx.ID,
			"agent_id":      ctx.AgentID,
			"parent_id":     ctx.ParentID,
			"created_at":    ctx.CreatedAt.Format(time.RFC3339),
			"last_accessed": ctx.LastAccessed.Format(time.RFC3339),
			"state":         string(ctx.State),
			"memory_refs":   len(ctx.MemoryRefs),
			"tools":         len(ctx.Tools),
			"routes":        len(ctx.Routes),
		})
	}

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"contexts": result,
			"count":    len(result),
		},
	}, nil
}

// handleGetStats 处理获取统计信息请求
func (p *ContextPlugin) handleGetStats(ctx context.Context, params map[string]interface{}) (*plugin.ToolResult, error) {
	stats := p.manager.GetStats()

	return &plugin.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"total_contexts":    stats.TotalContexts,
			"active_contexts":   stats.ActiveContexts,
			"total_memory_refs": stats.TotalMemoryRefs,
			"total_tools":       stats.TotalTools,
			"total_routes":      stats.TotalRoutes,
			"created_at":        stats.CreatedAt.Format(time.RFC3339),
			"last_updated":      stats.LastUpdated.Format(time.RFC3339),
			"storage_size":      stats.StorageSize,
		},
	}, nil
}

// ============================================================================
// 事件订阅
// ============================================================================

// subscribeEvents 订阅相关事件
func (p *ContextPlugin) subscribeEvents() error {
	// 订阅智慧体创建事件
	p.manager.eventBus.Subscribe("agent.created", func(data map[string]interface{}) {
		agentID, ok := data["agent_id"].(string)
		if !ok {
			return
		}

		// 自动为新建的智慧体创建上下文
		_, err := p.manager.CreateContext(agentID, "", map[string]interface{}{
			"auto_created": true,
		})
		if err != nil {
			fmt.Printf("⚠️ 自动创建智慧体 %s 上下文失败: %v\n", agentID, err)
		} else {
			fmt.Printf("✅ 自动为智慧体 %s 创建了上下文\n", agentID)
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

		// 将工具添加到智慧体上下文
		ctx, err := p.manager.GetContext(agentID)
		if err != nil {
			return
		}

		// 检查是否已存在
		for _, existingTool := range ctx.Tools {
			if existingTool == toolName {
				return
			}
		}

		// 添加工具
		ctx.Tools = append(ctx.Tools, toolName)
		if err := p.manager.UpdateContext(agentID, map[string]interface{}{
			"tools": ctx.Tools,
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

		// 将路由添加到智慧体上下文
		ctx, err := p.manager.GetContext(agentID)
		if err != nil {
			return
		}

		// 检查是否已存在
		for _, existingRoute := range ctx.Routes {
			if existingRoute == routeName {
				return
			}
		}

		// 添加路由
		ctx.Routes = append(ctx.Routes, routeName)
		if err := p.manager.UpdateContext(agentID, map[string]interface{}{
			"routes": ctx.Routes,
		}); err != nil {
			fmt.Printf("⚠️ 更新智慧体 %s 路由列表失败: %v\n", agentID, err)
		}
	})

	fmt.Println("✅ 上下文管理插件事件订阅完成")
	return nil
}
