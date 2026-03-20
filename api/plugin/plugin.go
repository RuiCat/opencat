package plugin

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// ============================================================================
// 兼容层：旧版插件系统
// ============================================================================

// PluginHandle 插件句柄（兼容旧版本）
type PluginHandle struct {
	*PluginInfo                          // 继承
	Handle      map[string]reflect.Value // 内部函数
}

// LegacyPluginManager 旧版插件管理器（兼容旧版本）
type LegacyPluginManager struct {
	mu      sync.RWMutex
	handles []PluginHandle // 插件列表
}

// NewLegacyPluginManager 创建新的旧版插件管理器
func NewLegacyPluginManager() *LegacyPluginManager {
	return &LegacyPluginManager{
		handles: make([]PluginHandle, 0),
	}
}

// AddPlugin 添加插件
func (pm *LegacyPluginManager) AddPlugin(info *PluginInfo, handle map[string]reflect.Value) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.handles = append(pm.handles, PluginHandle{
		PluginInfo: info,
		Handle:     handle,
	})
}

// GetPlugin 获取插件
func (pm *LegacyPluginManager) GetPlugin(name string) (*PluginInfo, map[string]reflect.Value, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, handle := range pm.handles {
		if handle.Name == name {
			return handle.PluginInfo, handle.Handle, true
		}
	}
	return nil, nil, false
}

// ListPlugins 列出所有插件
func (pm *LegacyPluginManager) ListPlugins() []*PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	infos := make([]*PluginInfo, len(pm.handles))
	for i, handle := range pm.handles {
		infos[i] = handle.PluginInfo
	}
	return infos
}

// RemovePlugin 移除插件
func (pm *LegacyPluginManager) RemovePlugin(name string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, handle := range pm.handles {
		if handle.Name == name {
			pm.handles = append(pm.handles[:i], pm.handles[i+1:]...)
			return true
		}
	}
	return false
}

// ============================================================================
// 基础插件实现
// ============================================================================

// BasePlugin 基础插件实现
type BasePlugin struct {
	info    *PluginInfo
	enabled bool
	tools   map[string]ToolHandler
	events  map[string][]EventHandler
	router  Router
	mu      sync.RWMutex
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(info *PluginInfo, router Router) *BasePlugin {
	return &BasePlugin{
		info:    info,
		enabled: true,
		tools:   make(map[string]ToolHandler),
		events:  make(map[string][]EventHandler),
		router:  router,
	}
}

// GetInfo 获取插件信息
func (p *BasePlugin) GetInfo() *PluginInfo {
	return p.info
}

// Init 初始化插件
func (p *BasePlugin) Init(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = true
	return nil
}

// Shutdown 关闭插件
func (p *BasePlugin) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	p.tools = make(map[string]ToolHandler)
	p.events = make(map[string][]EventHandler)
	return nil
}

// Enable 启用插件
func (p *BasePlugin) Enable() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = true
	return nil
}

// Disable 禁用插件
func (p *BasePlugin) Disable() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	return nil
}

// IsEnabled 检查插件是否启用
func (p *BasePlugin) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.enabled
}

// RegisterTool 注册工具
func (p *BasePlugin) RegisterTool(tool ToolDefinition, handler ToolHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.tools[tool.Name]; exists {
		return ErrToolAlreadyRegistered
	}

	p.tools[tool.Name] = handler
	return nil
}

// UnregisterTool 注销工具
func (p *BasePlugin) UnregisterTool(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.tools[name]; !exists {
		return ErrToolNotFound
	}

	delete(p.tools, name)
	return nil
}

// ListTools 列出工具
func (p *BasePlugin) ListTools() []ToolDefinition {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(p.tools))
	for name := range p.tools {
		// 这里简化处理，实际应该存储完整的 ToolDefinition
		tools = append(tools, ToolDefinition{
			Name:        name,
			Description: "工具描述",
			Enabled:     true,
		})
	}
	return tools
}

// CallTool 调用工具
func (p *BasePlugin) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	p.mu.RLock()
	handler, exists := p.tools[name]
	p.mu.RUnlock()

	if !exists {
		return nil, ErrToolNotFound
	}

	if !p.IsEnabled() {
		return &ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	return handler(ctx, args)
}

// EmitEvent 发送事件
func (p *BasePlugin) EmitEvent(ctx context.Context, event *Event) error {
	if p.router != nil {
		return p.router.PublishEvent(event.Type, event.Data)
	}
	return nil
}

// SubscribeEvent 订阅事件
func (p *BasePlugin) SubscribeEvent(eventType string, handler EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.events[eventType] = append(p.events[eventType], handler)
	return nil
}

// UnsubscribeEvent 取消订阅事件
func (p *BasePlugin) UnsubscribeEvent(eventType string, handler EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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
func (p *BasePlugin) CheckDependencies(ctx context.Context) error {
	deps := p.GetDependencies()
	if len(deps) == 0 {
		return nil
	}

	// 检查路由中是否存在这些函数
	if p.router == nil {
		return &PluginError{
			PluginName: p.info.Name,
			Message:    "路由未设置，无法检查依赖",
			Cause:      ErrDependencyMissing,
		}
	}

	for _, dep := range deps {
		// 检查函数是否存在
		if !p.router.HasFunction(dep) {
			return &PluginError{
				PluginName: p.info.Name,
				Message:    fmt.Sprintf("依赖函数不存在: %s", dep),
				Cause:      ErrDependencyMissing,
			}
		}
	}

	return nil
}

// GetDependencies 获取插件依赖
func (p *BasePlugin) GetDependencies() []string {
	if p.info == nil {
		return []string{}
	}
	return p.info.Dependencies
}

// HandleEvent 处理事件（内部使用）
func (p *BasePlugin) HandleEvent(event *Event) {
	p.mu.RLock()
	handlers, exists := p.events[event.Type]
	p.mu.RUnlock()

	if !exists {
		return
	}

	for _, handler := range handlers {
		go handler(event)
	}
}
