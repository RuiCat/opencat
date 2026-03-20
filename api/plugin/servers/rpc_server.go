package servers

import (
	"api/plugin"
	"api/router"
	"context"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// ============================================================================
// RPC 服务器插件
// ============================================================================

// RPCServerPlugin RPC 服务器插件
type RPCServerPlugin struct {
	*plugin.BasePlugin
	router    *router.Router
	server    *rpc.Server
	listener  net.Listener
	port      int
	mu        sync.RWMutex
	running   bool
	rpcClient *rpc.Client
}

// NewRPCServerPlugin 创建新的 RPC 服务器插件
func NewRPCServerPlugin(router *router.Router, port int) *RPCServerPlugin {
	info := &plugin.PluginInfo{
		Name:        "RPCServer",
		Version:     "1.0.0",
		Description: "RPC 服务器插件，提供远程过程调用支持",
		Author:      "opencat",
		CreatedAt:   time.Now(),
		Metadata: map[string]string{
			"category": "server",
			"type":     "rpc",
			"protocol": "gob",
		},
	}

	// 创建基础插件
	basePlugin := plugin.NewBasePlugin(info, nil)

	// 创建 RPC 服务器插件
	plugin := &RPCServerPlugin{
		BasePlugin: basePlugin,
		router:     router,
		port:       port,
		running:    false,
	}

	return plugin
}

// Init 初始化插件
func (p *RPCServerPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 注册 RPC 相关函数到 router
	if err := p.registerFunctions(); err != nil {
		return fmt.Errorf("注册函数失败: %w", err)
	}

	fmt.Printf("RPC 服务器插件初始化完成，端口: %d\n", p.port)
	return nil
}

// Shutdown 关闭插件
func (p *RPCServerPlugin) Shutdown(ctx context.Context) error {
	// 停止 RPC 服务器
	if p.running {
		if err := p.stopServer(); err != nil {
			fmt.Printf("停止 RPC 服务器失败: %v\n", err)
		}
	}

	// 关闭 RPC 客户端连接
	if p.rpcClient != nil {
		p.rpcClient.Close()
		p.rpcClient = nil
	}

	// 注销函数
	p.unregisterFunctions()

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// registerFunctions 注册函数到 router
func (p *RPCServerPlugin) registerFunctions() error {
	functions := []*router.Function{
		{
			Name:        "rpc.start",
			Description: "启动 RPC 服务器",
			Namespace:   "rpc",
			Handler:     p.handleStart,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "rpc.stop",
			Description: "停止 RPC 服务器",
			Namespace:   "rpc",
			Handler:     p.handleStop,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "rpc.status",
			Description: "获取 RPC 服务器状态",
			Namespace:   "rpc",
			Handler:     p.handleStatus,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "rpc.health",
			Description: "RPC 服务器健康检查",
			Namespace:   "rpc",
			Handler:     p.handleHealth,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "rpc.call",
			Description: "通过 RPC 调用远程函数",
			Namespace:   "rpc",
			Handler:     p.handleCall,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
	}

	for _, fn := range functions {
		fn.Stats = &router.FunctionStats{}
		if err := p.router.Register(fn); err != nil {
			return fmt.Errorf("注册函数 %s 失败: %w", fn.Name, err)
		}
		fmt.Printf("注册函数: %s\n", fn.Name)
	}

	return nil
}

// unregisterFunctions 注销函数
func (p *RPCServerPlugin) unregisterFunctions() {
	functions := []string{
		"rpc.start",
		"rpc.stop",
		"rpc.status",
		"rpc.health",
		"rpc.call",
	}

	for _, name := range functions {
		if err := p.router.Unregister(name); err != nil {
			fmt.Printf("注销函数 %s 失败: %v\n", name, err)
		} else {
			fmt.Printf("注销函数: %s\n", name)
		}
	}
}

// ============================================================================
// RPC 服务器实现
// ============================================================================

// startServer 启动 RPC 服务器
func (p *RPCServerPlugin) startServer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("RPC 服务器已在运行")
	}

	// 创建 RPC 服务器
	p.server = rpc.NewServer()

	// 注册 RPC 服务
	rpcService := &RPCService{plugin: p}
	if err := p.server.Register(rpcService); err != nil {
		return fmt.Errorf("注册 RPC 服务失败: %w", err)
	}

	// 启动监听
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		return fmt.Errorf("启动监听失败: %w", err)
	}

	p.listener = listener

	// 启动服务器
	go func() {
		fmt.Printf("RPC 服务器启动在端口 %d\n", p.port)
		p.server.Accept(listener)
	}()

	p.running = true
	return nil
}

// stopServer 停止 RPC 服务器
func (p *RPCServerPlugin) stopServer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.listener == nil {
		return fmt.Errorf("RPC 服务器未运行")
	}

	// 关闭监听器
	if err := p.listener.Close(); err != nil {
		return fmt.Errorf("关闭监听器失败: %w", err)
	}

	p.running = false
	p.listener = nil
	p.server = nil
	return nil
}

// connectClient 连接到 RPC 服务器（用于本地测试）
func (p *RPCServerPlugin) connectClient() error {
	if p.rpcClient != nil {
		return nil
	}

	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", p.port))
	if err != nil {
		return fmt.Errorf("连接 RPC 服务器失败: %w", err)
	}

	p.rpcClient = client
	return nil
}

// ============================================================================
// 函数处理器
// ============================================================================

// handleStart 处理 rpc.start 函数调用
func (p *RPCServerPlugin) handleStart(ctx *router.Context, block *router.DataBlock) *router.Result {
	if err := p.startServer(); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "RPC_START_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message":  "RPC 服务器已启动",
			"port":     p.port,
			"running":  true,
			"protocol": "gob",
		},
		TraceID: block.TraceID,
	}
}

// handleStop 处理 rpc.stop 函数调用
func (p *RPCServerPlugin) handleStop(ctx *router.Context, block *router.DataBlock) *router.Result {
	if err := p.stopServer(); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "RPC_STOP_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message": "RPC 服务器已停止",
			"port":    p.port,
			"running": false,
		},
		TraceID: block.TraceID,
	}
}

// handleStatus 处理 rpc.status 函数调用
func (p *RPCServerPlugin) handleStatus(ctx *router.Context, block *router.DataBlock) *router.Result {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"running":  running,
			"port":     p.port,
			"uptime":   time.Since(p.GetInfo().CreatedAt).String(),
			"protocol": "gob",
		},
		TraceID: block.TraceID,
	}
}

// handleHealth 处理 rpc.health 函数调用
func (p *RPCServerPlugin) handleHealth(ctx *router.Context, block *router.DataBlock) *router.Result {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	status := "healthy"
	if !running {
		status = "stopped"
	}

	// 如果服务器在运行，尝试连接测试
	if running {
		if err := p.connectClient(); err != nil {
			status = "unhealthy"
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"status":   status,
			"running":  running,
			"port":     p.port,
			"time":     time.Now().Format(time.RFC3339),
			"protocol": "gob",
		},
		TraceID: block.TraceID,
	}
}

// handleCall 处理 rpc.call 函数调用
func (p *RPCServerPlugin) handleCall(ctx *router.Context, block *router.DataBlock) *router.Result {
	// 获取参数
	functionName, ok := block.Payload["function"].(string)
	if !ok {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "INVALID_PARAMETER",
				Message: "参数 function 必须是字符串",
			},
			TraceID: block.TraceID,
		}
	}

	args, _ := block.Payload["args"].(map[string]interface{})

	// 检查服务器是否运行
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "RPC_SERVER_NOT_RUNNING",
				Message: "RPC 服务器未运行",
			},
			TraceID: block.TraceID,
		}
	}

	// 连接到 RPC 服务器
	if err := p.connectClient(); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "RPC_CONNECTION_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	// 准备 RPC 请求
	req := RPCRequest{
		Function: functionName,
		Args:     args,
	}

	var reply RPCResponse

	// 调用 RPC
	if err := p.rpcClient.Call("RPCService.CallFunction", req, &reply); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "RPC_CALL_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: reply.Success,
		Data:    reply.Data,
		Error: func() *router.ErrorInfo {
			if reply.Error != "" {
				return &router.ErrorInfo{
					Code:    "RPC_ERROR",
					Message: reply.Error,
				}
			}
			return nil
		}(),
		TraceID: block.TraceID,
	}
}

// ============================================================================
// RPC 服务实现
// ============================================================================

// RPCService RPC 服务
type RPCService struct {
	plugin *RPCServerPlugin
}

// RPCRequest RPC 请求
type RPCRequest struct {
	Function string                 `json:"function"`
	Args     map[string]interface{} `json:"args"`
}

// RPCResponse RPC 响应
type RPCResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CallFunction 调用函数 RPC 方法
func (s *RPCService) CallFunction(req RPCRequest, reply *RPCResponse) error {
	// 创建 router 上下文
	routerCtx := router.NewContext("rpc", "rpc-client", s.plugin.router)
	block := router.NewDataBlock(req.Function, req.Args)

	// 调用函数
	result := s.plugin.router.Call(routerCtx, block)

	*reply = RPCResponse{
		Success: result.Success,
		Data:    result.Data,
		Error: func() string {
			if result.Error != nil {
				return result.Error.Message
			}
			return ""
		}(),
	}

	return nil
}

// ListFunctions 列出函数 RPC 方法
func (s *RPCService) ListFunctions(args struct{}, reply *[]string) error {
	functions := s.plugin.router.List("")
	names := make([]string, len(functions))
	for i, fn := range functions {
		names[i] = fn.Name
	}
	*reply = names
	return nil
}

// HealthCheck 健康检查 RPC 方法
func (s *RPCService) HealthCheck(args struct{}, reply *map[string]interface{}) error {
	s.plugin.mu.RLock()
	running := s.plugin.running
	s.plugin.mu.RUnlock()

	*reply = map[string]interface{}{
		"status":  "healthy",
		"running": running,
		"port":    s.plugin.port,
		"time":    time.Now().Format(time.RFC3339),
	}

	return nil
}
