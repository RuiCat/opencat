package plugin

import (
	"api/router"
	"context"
	"fmt"
	"time"
)

// ============================================================================
// Router 适配器 - 连接 router.Router 和 plugin.Router 接口
// ============================================================================

// RouterAdapter 将 router.Router 适配为 plugin.Router 接口
type RouterAdapter struct {
	router        *router.Router
	pluginManager *Manager
}

// NewRouterAdapter 创建新的 Router 适配器
func NewRouterAdapter(r *router.Router) *RouterAdapter {
	adapter := &RouterAdapter{
		router: r,
	}
	// 创建插件管理器并关联到适配器
	adapter.pluginManager = NewManager(adapter)
	return adapter
}

// GetRouter 获取底层 router.Router
func (ra *RouterAdapter) GetRouter() *router.Router {
	return ra.router
}

// ============================================================================
// 实现 plugin.Router 接口
// ============================================================================

// CallFunction 调用 router 上注册的函数
func (ra *RouterAdapter) CallFunction(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	if ra.router == nil {
		return nil, fmt.Errorf("router 未设置")
	}

	// 创建执行上下文
	routerCtx := router.NewContext("plugin", "plugin-caller", ra.router)

	// 设置超时
	if deadline, ok := ctx.Deadline(); ok {
		routerCtx.WithTimeout(time.Until(deadline))
	}

	// 创建数据块
	block := router.NewDataBlock(name, args)

	// 调用函数
	result := ra.router.Call(routerCtx, block)

	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("函数调用失败")
	}

	return result.Data, nil
}

// RegisterFunction 注册函数到 router
func (ra *RouterAdapter) RegisterFunction(name string, handler interface{}) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}

	// 尝试转换 handler 为 router.FunctionHandler
	var routerHandler router.FunctionHandler

	switch h := handler.(type) {
	case router.FunctionHandler:
		routerHandler = h
	case func(*router.Context, *router.DataBlock) *router.Result:
		routerHandler = h
	case func(context.Context, map[string]interface{}) (interface{}, error):
		// 包装通用处理器
		routerHandler = ra.wrapGenericHandler(h)
	case ToolHandler:
		// 包装 ToolHandler
		routerHandler = ra.wrapToolHandler(h)
	default:
		return fmt.Errorf("不支持的 handler 类型: %T", handler)
	}

	// 创建函数并注册
	fn := &router.Function{
		Name:        name,
		Description: fmt.Sprintf("插件函数: %s", name),
		Namespace:   extractNamespace(name),
		Handler:     routerHandler,
		Builtin:     false,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	return ra.router.Register(fn)
}

// UnregisterFunction 从 router 注销函数
func (ra *RouterAdapter) UnregisterFunction(name string) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}
	return ra.router.Unregister(name)
}

// ListFunctions 列出 router 上所有函数
func (ra *RouterAdapter) ListFunctions() []string {
	if ra.router == nil {
		return []string{}
	}

	functions := ra.router.List("")
	names := make([]string, len(functions))
	for i, fn := range functions {
		names[i] = fn.Name
	}
	return names
}

// HasFunction 检查函数是否存在
func (ra *RouterAdapter) HasFunction(name string) bool {
	if ra.router == nil {
		return false
	}

	_, err := ra.router.GetFunction(name)
	return err == nil
}

// PublishEvent 发布事件到 router
func (ra *RouterAdapter) PublishEvent(eventType string, data interface{}) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}

	ra.router.Publish(eventType, data)
	return nil
}

// SubscribeEvent 订阅 router 事件
func (ra *RouterAdapter) SubscribeEvent(eventType string, handler interface{}) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}

	// 转换 handler 为 router.EventHandler
	var routerHandler router.EventHandler

	switch h := handler.(type) {
	case router.EventHandler:
		routerHandler = h
	case func(string, interface{}):
		routerHandler = h
	case EventHandler:
		// 包装 plugin.EventHandler
		routerHandler = func(event string, data interface{}) {
			h(&Event{
				Type:      event,
				Name:      event,
				Data:      data,
				Timestamp: time.Now(),
				Source:    "router",
			})
		}
	default:
		return fmt.Errorf("不支持的 handler 类型: %T", handler)
	}

	ra.router.Subscribe(eventType, routerHandler)
	return nil
}

// RegisterPlugin 注册插件到管理器
func (ra *RouterAdapter) RegisterPlugin(plugin Plugin) error {
	if ra.pluginManager == nil {
		return fmt.Errorf("插件管理器未初始化")
	}
	return ra.pluginManager.LoadPlugin(plugin.GetInfo().Name, plugin)
}

// UnregisterPlugin 注销插件
func (ra *RouterAdapter) UnregisterPlugin(name string) error {
	if ra.pluginManager == nil {
		return fmt.Errorf("插件管理器未初始化")
	}
	return ra.pluginManager.UnloadPlugin(name)
}

// GetPluginManager 获取插件管理器
func (ra *RouterAdapter) GetPluginManager() PluginManagerInterface {
	return ra.pluginManager
}

// RegisterManagementInterface 注册管理接口
func (ra *RouterAdapter) RegisterManagementInterface(name string, handler interface{}) error {
	if ra.pluginManager == nil {
		return fmt.Errorf("插件管理器未初始化")
	}
	ra.pluginManager.management[name] = handler
	return nil
}

// GetManagementInterface 获取管理接口
func (ra *RouterAdapter) GetManagementInterface(name string) (interface{}, error) {
	if ra.pluginManager == nil {
		return nil, fmt.Errorf("插件管理器未初始化")
	}
	handler, exists := ra.pluginManager.management[name]
	if !exists {
		return nil, fmt.Errorf("管理接口未找到: %s", name)
	}
	return handler, nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// wrapGenericHandler 包装通用处理器为 router.FunctionHandler
func (ra *RouterAdapter) wrapGenericHandler(handler func(context.Context, map[string]interface{}) (interface{}, error)) router.FunctionHandler {
	return func(ctx *router.Context, block *router.DataBlock) *router.Result {
		// 创建 context.Context
		goCtx := context.Background()

		// 调用处理器
		data, err := handler(goCtx, block.Payload)
		if err != nil {
			return &router.Result{
				Success: false,
				Error: &router.ErrorInfo{
					Code:    "HANDLER_ERROR",
					Message: err.Error(),
				},
				TraceID: block.TraceID,
			}
		}

		return &router.Result{
			Success: true,
			Data:    data,
			TraceID: block.TraceID,
		}
	}
}

// wrapToolHandler 包装 ToolHandler 为 router.FunctionHandler
func (ra *RouterAdapter) wrapToolHandler(handler ToolHandler) router.FunctionHandler {
	return func(ctx *router.Context, block *router.DataBlock) *router.Result {
		// 创建 context.Context
		goCtx := context.Background()

		// 调用工具处理器
		toolResult, err := handler(goCtx, block.Payload)
		if err != nil {
			return &router.Result{
				Success: false,
				Error: &router.ErrorInfo{
					Code:    "TOOL_ERROR",
					Message: err.Error(),
				},
				TraceID: block.TraceID,
			}
		}

		if toolResult == nil {
			return &router.Result{
				Success: true,
				TraceID: block.TraceID,
			}
		}

		return &router.Result{
			Success: toolResult.Success,
			Data:    toolResult.Data,
			Error: func() *router.ErrorInfo {
				if toolResult.Error != "" {
					return &router.ErrorInfo{
						Code:    "TOOL_RESULT_ERROR",
						Message: toolResult.Error,
					}
				}
				return nil
			}(),
			TraceID: block.TraceID,
		}
	}
}

// extractNamespace 从函数名提取命名空间
func extractNamespace(name string) string {
	for i, c := range name {
		if c == '.' {
			return name[:i]
		}
	}
	return "default"
}

// ============================================================================
// 函数注册辅助
// ============================================================================

// RegisterFunctionWithInfo 注册带完整信息的函数
func (ra *RouterAdapter) RegisterFunctionWithInfo(fn *router.Function) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}
	return ra.router.Register(fn)
}

// GetFunction 获取函数信息
func (ra *RouterAdapter) GetFunction(name string) (*router.Function, error) {
	if ra.router == nil {
		return nil, fmt.Errorf("router 未设置")
	}
	return ra.router.GetFunction(name)
}

// EnableFunction 启用函数
func (ra *RouterAdapter) EnableFunction(name string) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}
	return ra.router.Enable(name)
}

// DisableFunction 禁用函数
func (ra *RouterAdapter) DisableFunction(name string) error {
	if ra.router == nil {
		return fmt.Errorf("router 未设置")
	}
	return ra.router.Disable(name)
}

// GetRouterStats 获取路由统计信息
func (ra *RouterAdapter) GetRouterStats() *router.RouterStats {
	if ra.router == nil {
		return nil
	}
	return ra.router.GetStats()
}
