package system

import (
	"context"
	"time"

	"api/router"
)

// ============================================================================
// 路由管理器接口
// ============================================================================

// RouterManager 路由管理器接口
type RouterManager interface {
	// 函数管理
	RegisterFunction(ctx context.Context, name, description, namespace string) error
	UnregisterFunction(ctx context.Context, name string) error
	ListFunctions(ctx context.Context, namespace string) ([]*router.Function, error)
	EnableFunction(ctx context.Context, name string) error
	DisableFunction(ctx context.Context, name string) error

	// 路由信息
	GetRouterStats(ctx context.Context) (*router.RouterStats, error)
	GetFunctionStats(ctx context.Context, name string) (*router.FunctionStats, error)
	QueryAuditLogs(ctx context.Context, filter router.AuditLogFilter) ([]router.AuditLog, error)

	// 触发器管理
	RegisterTrigger(ctx context.Context, trigger *router.Trigger) error
	UnregisterTrigger(ctx context.Context, id string) error
	ListTriggers(ctx context.Context) ([]*router.Trigger, error)
	EnableTrigger(ctx context.Context, id string) error
	DisableTrigger(ctx context.Context, id string) error

	// 事件管理
	PublishCustomEvent(ctx context.Context, eventName string, data map[string]interface{}) error
}

// ============================================================================
// 路由管理器实现
// ============================================================================

// routerManagerImpl 路由管理器实现
type routerManagerImpl struct {
	router *router.Router
}

// NewRouterManager 创建路由管理器
func NewRouterManager(r *router.Router) RouterManager {
	return &routerManagerImpl{
		router: r,
	}
}

// RegisterFunction 注册函数
func (rm *routerManagerImpl) RegisterFunction(ctx context.Context, name, description, namespace string) error {
	// 注意：这个实现是简化的，实际需要更复杂的逻辑
	// 特别是处理器的设置需要额外机制
	return &SystemError{
		Code:    "NOT_IMPLEMENTED",
		Message: "router.register 需要通过其他机制设置处理器",
	}
}

// UnregisterFunction 注销函数
func (rm *routerManagerImpl) UnregisterFunction(ctx context.Context, name string) error {
	return rm.router.Unregister(name)
}

// ListFunctions 列出函数
func (rm *routerManagerImpl) ListFunctions(ctx context.Context, namespace string) ([]*router.Function, error) {
	return rm.router.List(namespace), nil
}

// EnableFunction 启用函数
func (rm *routerManagerImpl) EnableFunction(ctx context.Context, name string) error {
	return rm.router.Enable(name)
}

// DisableFunction 禁用函数
func (rm *routerManagerImpl) DisableFunction(ctx context.Context, name string) error {
	return rm.router.Disable(name)
}

// GetRouterStats 获取路由统计
func (rm *routerManagerImpl) GetRouterStats(ctx context.Context) (*router.RouterStats, error) {
	return rm.router.GetStats(), nil
}

// GetFunctionStats 获取函数统计
func (rm *routerManagerImpl) GetFunctionStats(ctx context.Context, name string) (*router.FunctionStats, error) {
	return rm.router.GetFunctionStats(name)
}

// QueryAuditLogs 查询审计日志
func (rm *routerManagerImpl) QueryAuditLogs(ctx context.Context, filter router.AuditLogFilter) ([]router.AuditLog, error) {
	return rm.router.QueryLogs(filter), nil
}

// RegisterTrigger 注册触发器
func (rm *routerManagerImpl) RegisterTrigger(ctx context.Context, trigger *router.Trigger) error {
	return rm.router.RegisterTrigger(trigger)
}

// UnregisterTrigger 注销触发器
func (rm *routerManagerImpl) UnregisterTrigger(ctx context.Context, id string) error {
	return rm.router.UnregisterTrigger(id)
}

// ListTriggers 列出触发器
func (rm *routerManagerImpl) ListTriggers(ctx context.Context) ([]*router.Trigger, error) {
	return rm.router.ListTriggers(), nil
}

// EnableTrigger 启用触发器
func (rm *routerManagerImpl) EnableTrigger(ctx context.Context, id string) error {
	return rm.router.EnableTrigger(id)
}

// DisableTrigger 禁用触发器
func (rm *routerManagerImpl) DisableTrigger(ctx context.Context, id string) error {
	return rm.router.DisableTrigger(id)
}

// PublishCustomEvent 发布自定义事件
func (rm *routerManagerImpl) PublishCustomEvent(ctx context.Context, eventName string, data map[string]interface{}) error {
	return rm.router.PublishCustomEvent(eventName, data)
}

// ============================================================================
// 路由函数注册
// ============================================================================

// registerRouterFunctions 注册路由管理函数
func registerRouterFunctions(r *router.Router, _ *SystemManager) error {
	// 创建路由管理器
	routerMgr := NewRouterManager(r)

	// 注册函数函数
	r.Register(&router.Function{
		Name:        "system.router.register",
		Description: "注册新函数到路由",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "函数名称",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "函数描述",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "函数命名空间",
				},
			},
			"required": []string{"name", "description", "namespace"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			name, ok := block.Payload["name"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 name 必须是字符串",
					},
				}
			}

			description, ok := block.Payload["description"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 description 必须是字符串",
					},
				}
			}

			namespace, ok := block.Payload["namespace"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 namespace 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			err := routerMgr.RegisterFunction(context.Background(), name, description, namespace)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "REGISTER_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"message": "函数注册成功",
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 注销函数函数
	r.Register(&router.Function{
		Name:        "system.router.unregister",
		Description: "从路由注销函数",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "函数名称",
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			name, ok := block.Payload["name"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 name 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			err := routerMgr.UnregisterFunction(context.Background(), name)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "UNREGISTER_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"message": "函数已注销",
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 列出函数函数
	r.Register(&router.Function{
		Name:        "system.router.list",
		Description: "列出路由中的所有函数",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "命名空间过滤",
				},
			},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			namespace := ""
			if ns, ok := block.Payload["namespace"].(string); ok {
				namespace = ns
			}

			// 使用 context.Background() 替代 ctx.Context
			functions, err := routerMgr.ListFunctions(context.Background(), namespace)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "LIST_FAILED",
						Message: err.Error(),
					},
				}
			}

			result := make([]map[string]interface{}, len(functions))
			for i, fn := range functions {
				result[i] = map[string]interface{}{
					"name":        fn.Name,
					"description": fn.Description,
					"namespace":   fn.Namespace,
					"builtin":     fn.Builtin,
					"enabled":     fn.Enabled,
					"created_at":  fn.CreatedAt,
					"stats":       fn.Stats,
				}
			}

			return &router.Result{
				Success: true,
				Data:    result,
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 启用函数函数
	r.Register(&router.Function{
		Name:        "system.router.enable",
		Description: "启用函数",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "函数名称",
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			name, ok := block.Payload["name"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 name 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			err := routerMgr.EnableFunction(context.Background(), name)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "ENABLE_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"message": "函数已启用",
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 禁用函数函数
	r.Register(&router.Function{
		Name:        "system.router.disable",
		Description: "禁用函数",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "函数名称",
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			name, ok := block.Payload["name"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 name 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			err := routerMgr.DisableFunction(context.Background(), name)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "DISABLE_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"message": "函数已禁用",
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 获取路由统计函数
	r.Register(&router.Function{
		Name:        "system.router.stats",
		Description: "获取路由统计信息",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			// 使用 context.Background() 替代 ctx.Context
			stats, err := routerMgr.GetRouterStats(context.Background())
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "STATS_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data:    stats,
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 查询审计日志函数
	r.Register(&router.Function{
		Name:        "system.router.logs",
		Description: "查询路由审计日志",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "操作类型",
				},
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "代理ID",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "目标函数",
				},
				"success": map[string]interface{}{
					"type":        "boolean",
					"description": "是否成功",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量限制",
				},
			},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			filter := router.AuditLogFilter{}

			if action, ok := block.Payload["action"].(string); ok {
				filter.Action = action
			}
			if agentID, ok := block.Payload["agent_id"].(string); ok {
				filter.AgentID = agentID
			}
			if target, ok := block.Payload["target"].(string); ok {
				filter.Target = target
			}
			if success, ok := block.Payload["success"].(bool); ok {
				filter.Success = &success
			}
			if limit, ok := block.Payload["limit"].(float64); ok {
				filter.Limit = int(limit)
			}

			// 使用 context.Background() 替代 ctx.Context
			logs, err := routerMgr.QueryAuditLogs(context.Background(), filter)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "LOGS_FAILED",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data:    logs,
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	return nil
}
