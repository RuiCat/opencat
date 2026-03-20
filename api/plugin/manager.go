package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// 插件管理器实现
// ============================================================================

// Manager 插件管理器实现
type Manager struct {
	mu            sync.RWMutex
	plugins       map[string]Plugin
	tools         map[string]*toolEntry
	eventHandlers map[string][]EventHandler
	router        Router
	management    map[string]interface{} // 管理接口注册表
}

// toolEntry 工具条目
type toolEntry struct {
	pluginName string
	tool       ToolDefinition
	handler    ToolHandler
}

// NewManager 创建新的插件管理器
func NewManager(router Router) *Manager {
	return &Manager{
		plugins:       make(map[string]Plugin),
		tools:         make(map[string]*toolEntry),
		eventHandlers: make(map[string][]EventHandler),
		router:        router,
		management:    make(map[string]interface{}),
	}
}

// LoadPlugin 加载插件
func (m *Manager) LoadPlugin(name string, plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; exists {
		return ErrPluginAlreadyLoaded
	}

	// 检查插件依赖
	if err := m.checkPluginDependencies(plugin); err != nil {
		return fmt.Errorf("插件依赖检查失败: %w", err)
	}

	// 初始化插件
	ctx := context.Background()
	if err := plugin.Init(ctx); err != nil {
		return fmt.Errorf("插件初始化失败: %w", err)
	}

	// 注册插件工具
	tools := plugin.ListTools()
	for _, tool := range tools {
		toolName := fmt.Sprintf("%s.%s", name, tool.Name)
		m.tools[toolName] = &toolEntry{
			pluginName: name,
			tool:       tool,
			handler: func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
				return plugin.CallTool(ctx, tool.Name, args)
			},
		}
	}

	// 注册插件管理接口到路由
	if m.router != nil {
		managementName := fmt.Sprintf("plugin.management.%s", name)
		if err := m.router.RegisterManagementInterface(managementName, plugin); err != nil {
			// 记录错误但不阻止插件加载
			fmt.Printf("注册插件管理接口失败: %v\n", err)
		}
	}

	m.plugins[name] = plugin

	// 发布插件加载事件
	m.emitEvent(ctx, &Event{
		Type:      "plugin.loaded",
		Name:      name,
		Data:      plugin.GetInfo(),
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	})

	return nil
}

// UnloadPlugin 卸载插件
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	// 关闭插件
	ctx := context.Background()
	if err := plugin.Shutdown(ctx); err != nil {
		return fmt.Errorf("插件关闭失败: %w", err)
	}

	// 移除插件工具
	for toolName, entry := range m.tools {
		if entry.pluginName == name {
			delete(m.tools, toolName)
		}
	}

	// 从路由移除管理接口
	if m.router != nil {
		// 注意：这里假设路由有对应的注销方法
		// 实际实现中可能需要调用路由的UnregisterManagementInterface
		// managementName := fmt.Sprintf("plugin.management.%s", name)
	}

	delete(m.plugins, name)

	// 发布插件卸载事件
	m.emitEvent(ctx, &Event{
		Type:      "plugin.unloaded",
		Name:      name,
		Data:      plugin.GetInfo(),
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	})

	return nil
}

// GetPlugin 获取插件
func (m *Manager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return plugin, nil
}

// ListPlugins 列出所有插件
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		infos = append(infos, *plugin.GetInfo())
	}
	return infos
}

// RegisterTool 注册工具
func (m *Manager) RegisterTool(pluginName string, tool ToolDefinition, handler ToolHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	toolName := fmt.Sprintf("%s.%s", pluginName, tool.Name)
	if _, exists := m.tools[toolName]; exists {
		return ErrToolAlreadyRegistered
	}

	m.tools[toolName] = &toolEntry{
		pluginName: pluginName,
		tool:       tool,
		handler:    handler,
	}

	return nil
}

// UnregisterTool 注销工具
func (m *Manager) UnregisterTool(pluginName, toolName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullName := fmt.Sprintf("%s.%s", pluginName, toolName)
	if _, exists := m.tools[fullName]; !exists {
		return ErrToolNotFound
	}

	delete(m.tools, fullName)
	return nil
}

// ListTools 列出所有工具
func (m *Manager) ListTools() []ToolDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(m.tools))
	for _, entry := range m.tools {
		tools = append(tools, entry.tool)
	}
	return tools
}

// CallTool 调用工具
func (m *Manager) CallTool(ctx context.Context, pluginName, toolName string, args map[string]interface{}) (*ToolResult, error) {
	m.mu.RLock()
	fullName := fmt.Sprintf("%s.%s", pluginName, toolName)
	entry, exists := m.tools[fullName]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrToolNotFound
	}

	// 检查插件是否启用
	plugin, err := m.GetPlugin(pluginName)
	if err != nil {
		return nil, err
	}

	if !plugin.IsEnabled() {
		return &ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	return entry.handler(ctx, args)
}

// EmitEvent 发送事件
func (m *Manager) EmitEvent(ctx context.Context, event *Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 调用路由的事件发布
	if m.router != nil {
		return m.router.PublishEvent(event.Type, event.Data)
	}

	// 如果没有路由，直接调用本地处理器
	m.mu.RLock()
	handlers := m.eventHandlers[event.Type]
	m.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}

	return nil
}

// SubscribeEvent 订阅事件
func (m *Manager) SubscribeEvent(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
	return nil
}

// UnsubscribeEvent 取消订阅事件
func (m *Manager) UnsubscribeEvent(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handlers, exists := m.eventHandlers[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if &h == &handler {
			m.eventHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// EnablePlugin 启用插件
func (m *Manager) EnablePlugin(name string) error {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return err
	}

	if err := plugin.Enable(); err != nil {
		return err
	}

	// 发布插件启用事件
	ctx := context.Background()
	m.emitEvent(ctx, &Event{
		Type:      "plugin.enabled",
		Name:      name,
		Data:      plugin.GetInfo(),
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	})

	return nil
}

// DisablePlugin 禁用插件
func (m *Manager) DisablePlugin(name string) error {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return err
	}

	if err := plugin.Disable(); err != nil {
		return err
	}

	// 发布插件禁用事件
	ctx := context.Background()
	m.emitEvent(ctx, &Event{
		Type:      "plugin.disabled",
		Name:      name,
		Data:      plugin.GetInfo(),
		Timestamp: time.Now(),
		Source:    "plugin.manager",
	})

	return nil
}

// GetPluginStatus 获取插件状态
func (m *Manager) GetPluginStatus(name string) (bool, error) {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return false, err
	}

	return plugin.IsEnabled(), nil
}

// HandleRouterEvent 处理路由事件（供路由调用）
func (m *Manager) HandleRouterEvent(eventType string, data interface{}) {
	event := &Event{
		Type:      eventType,
		Name:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "router",
	}

	m.mu.RLock()
	handlers := m.eventHandlers[eventType]
	m.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// ============================================================================
// 依赖管理方法
// ============================================================================

// CheckPluginDependencies 检查插件依赖
func (m *Manager) CheckPluginDependencies(name string) error {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return plugin.CheckDependencies(ctx)
}

// GetPluginDependencies 获取插件依赖
func (m *Manager) GetPluginDependencies(name string) ([]string, error) {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	return plugin.GetDependencies(), nil
}

// checkPluginDependencies 内部依赖检查方法
func (m *Manager) checkPluginDependencies(plugin Plugin) error {
	deps := plugin.GetDependencies()
	if len(deps) == 0 {
		return nil
	}

	// 检查循环依赖
	if err := m.checkCircularDependencies(plugin.GetInfo().Name, deps); err != nil {
		return err
	}

	// 检查依赖是否存在
	for _, dep := range deps {
		// 检查是否已加载的插件
		if _, exists := m.plugins[dep]; exists {
			continue
		}

		// 检查路由中是否有对应的函数
		if m.router != nil && m.router.HasFunction(dep) {
			continue
		}

		return &PluginError{
			PluginName: plugin.GetInfo().Name,
			Message:    fmt.Sprintf("依赖缺失: %s", dep),
			Cause:      ErrDependencyMissing,
		}
	}

	return nil
}

// checkCircularDependencies 检查循环依赖
func (m *Manager) checkCircularDependencies(pluginName string, deps []string) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var check func(string) error
	check = func(name string) error {
		if stack[name] {
			return &PluginError{
				PluginName: pluginName,
				Message:    fmt.Sprintf("检测到循环依赖: %s", name),
				Cause:      ErrCircularDependency,
			}
		}

		if visited[name] {
			return nil
		}

		visited[name] = true
		stack[name] = true

		// 获取该插件的依赖
		plugin, exists := m.plugins[name]
		if exists {
			for _, dep := range plugin.GetDependencies() {
				if err := check(dep); err != nil {
					return err
				}
			}
		}

		delete(stack, name)
		return nil
	}

	for _, dep := range deps {
		if err := check(dep); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================================
// 事件掩盖机制
// ============================================================================

// emitEvent 发送事件（带掩盖机制）
func (m *Manager) emitEvent(ctx context.Context, event *Event) error {
	// 检查是否需要掩盖事件
	if m.shouldMaskEvent(event) {
		// 掩盖事件：转换为内部事件，不暴露给外部
		maskedEvent := &Event{
			Type:      "internal." + event.Type,
			Name:      event.Name,
			Data:      m.maskEventData(event.Data),
			Timestamp: event.Timestamp,
			Source:    "masked." + event.Source,
		}

		// 发送掩盖后的事件
		return m.EmitEvent(ctx, maskedEvent)
	}

	// 发送原始事件
	return m.EmitEvent(ctx, event)
}

// shouldMaskEvent 判断是否需要掩盖事件
func (m *Manager) shouldMaskEvent(event *Event) bool {
	// 需要掩盖的事件类型
	maskableEvents := map[string]bool{
		"plugin.load.error":     true,
		"plugin.unload.error":   true,
		"plugin.init.error":     true,
		"plugin.shutdown.error": true,
		"tool.call.error":       true,
		"dependency.error":      true,
	}

	return maskableEvents[event.Type]
}

// maskEventData 掩盖事件数据
func (m *Manager) maskEventData(data interface{}) interface{} {
	// 将错误信息转换为通用格式，隐藏具体细节
	return map[string]interface{}{
		"type":        "masked_error",
		"description": "发生内部错误，已记录到日志",
		"timestamp":   time.Now().Unix(),
		"severity":    "internal",
	}
}
