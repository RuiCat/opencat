package servers

import (
	"api/plugin"
	"api/router"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ============================================================================
// MCP 服务器插件
// ============================================================================

// MCPServerPlugin MCP 服务器插件
type MCPServerPlugin struct {
	*plugin.BasePlugin
	router  *router.Router
	server  *http.Server
	port    int
	mu      sync.RWMutex
	running bool
}

// NewMCPServerPlugin 创建新的 MCP 服务器插件
func NewMCPServerPlugin(router *router.Router, port int) *MCPServerPlugin {
	info := &plugin.PluginInfo{
		Name:        "MCPServer",
		Version:     "1.0.0",
		Description: "MCP 服务器插件，提供 Model Context Protocol 支持",
		Author:      "opencat",
		CreatedAt:   time.Now(),
		Metadata: map[string]string{
			"category": "server",
			"type":     "mcp",
			"protocol": "2024-11-05",
		},
	}

	// 创建基础插件
	basePlugin := plugin.NewBasePlugin(info, nil)

	// 创建 MCP 服务器插件
	plugin := &MCPServerPlugin{
		BasePlugin: basePlugin,
		router:     router,
		port:       port,
		running:    false,
	}

	return plugin
}

// Init 初始化插件
func (p *MCPServerPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 注册 MCP 相关函数到 router
	if err := p.registerFunctions(); err != nil {
		return fmt.Errorf("注册函数失败: %w", err)
	}

	fmt.Printf("MCP 服务器插件初始化完成，端口: %d\n", p.port)
	return nil
}

// Shutdown 关闭插件
func (p *MCPServerPlugin) Shutdown(ctx context.Context) error {
	// 停止 MCP 服务器
	if p.running {
		if err := p.stopServer(ctx); err != nil {
			fmt.Printf("停止 MCP 服务器失败: %v\n", err)
		}
	}

	// 注销函数
	p.unregisterFunctions()

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// registerFunctions 注册函数到 router
func (p *MCPServerPlugin) registerFunctions() error {
	functions := []*router.Function{
		{
			Name:        "mcp.start",
			Description: "启动 MCP 服务器",
			Namespace:   "mcp",
			Handler:     p.handleStart,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "mcp.stop",
			Description: "停止 MCP 服务器",
			Namespace:   "mcp",
			Handler:     p.handleStop,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "mcp.status",
			Description: "获取 MCP 服务器状态",
			Namespace:   "mcp",
			Handler:     p.handleStatus,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "mcp.tools.list",
			Description: "列出 MCP 可用工具",
			Namespace:   "mcp",
			Handler:     p.handleToolsList,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "mcp.health",
			Description: "MCP 服务器健康检查",
			Namespace:   "mcp",
			Handler:     p.handleHealth,
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
func (p *MCPServerPlugin) unregisterFunctions() {
	functions := []string{
		"mcp.start",
		"mcp.stop",
		"mcp.status",
		"mcp.tools.list",
		"mcp.health",
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
// MCP 服务器实现
// ============================================================================

// startServer 启动 MCP 服务器
func (p *MCPServerPlugin) startServer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("MCP 服务器已在运行")
	}

	// 创建路由器
	mux := http.NewServeMux()

	// 注册 MCP 协议端点
	mux.HandleFunc("/mcp/initialize", p.handleInitialize)
	mux.HandleFunc("/mcp/tools/list", p.handleToolsListAPI)
	mux.HandleFunc("/mcp/tools/call", p.handleToolsCall)
	mux.HandleFunc("/mcp/events", p.handleEvents)
	mux.HandleFunc("/mcp/health", p.handleHealthAPI)

	// 创建 HTTP 服务器
	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		fmt.Printf("MCP 服务器启动在端口 %d\n", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP 服务器错误: %v\n", err)
		}
	}()

	p.running = true
	return nil
}

// stopServer 停止 MCP 服务器
func (p *MCPServerPlugin) stopServer(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.server == nil {
		return fmt.Errorf("MCP 服务器未运行")
	}

	// 创建超时上下文
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 优雅关闭
	if err := p.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("MCP 服务器关闭失败: %w", err)
	}

	p.running = false
	p.server = nil
	return nil
}

// ============================================================================
// 函数处理器
// ============================================================================

// handleStart 处理 mcp.start 函数调用
func (p *MCPServerPlugin) handleStart(ctx *router.Context, block *router.DataBlock) *router.Result {
	if err := p.startServer(); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "MCP_START_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message":  "MCP 服务器已启动",
			"port":     p.port,
			"running":  true,
			"protocol": "2024-11-05",
		},
		TraceID: block.TraceID,
	}
}

// handleStop 处理 mcp.stop 函数调用
func (p *MCPServerPlugin) handleStop(ctx *router.Context, block *router.DataBlock) *router.Result {
	// 创建 context.Context
	goCtx := context.Background()
	if err := p.stopServer(goCtx); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "MCP_STOP_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message": "MCP 服务器已停止",
			"port":    p.port,
			"running": false,
		},
		TraceID: block.TraceID,
	}
}

// handleStatus 处理 mcp.status 函数调用
func (p *MCPServerPlugin) handleStatus(ctx *router.Context, block *router.DataBlock) *router.Result {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"running":  running,
			"port":     p.port,
			"uptime":   time.Since(p.GetInfo().CreatedAt).String(),
			"protocol": "2024-11-05",
		},
		TraceID: block.TraceID,
	}
}

// handleToolsList 处理 mcp.tools.list 函数调用
func (p *MCPServerPlugin) handleToolsList(ctx *router.Context, block *router.DataBlock) *router.Result {
	// 获取 router 中的所有函数
	functions := p.router.List("")

	// 转换为 MCP 工具格式
	tools := make([]map[string]interface{}, 0)
	for _, fn := range functions {
		// 只暴露特定命名空间的函数
		if fn.Namespace == "util" || fn.Namespace == "sys" || fn.Namespace == "fs" {
			tool := map[string]interface{}{
				"name":        fn.Name,
				"description": fn.Description,
				"enabled":     fn.Enabled,
				"namespace":   fn.Namespace,
			}
			tools = append(tools, tool)
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"tools": tools,
			"count": len(tools),
		},
		TraceID: block.TraceID,
	}
}

// handleHealth 处理 mcp.health 函数调用
func (p *MCPServerPlugin) handleHealth(ctx *router.Context, block *router.DataBlock) *router.Result {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	status := "healthy"
	if !running {
		status = "stopped"
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"status":   status,
			"running":  running,
			"port":     p.port,
			"time":     time.Now().Format(time.RFC3339),
			"protocol": "2024-11-05",
		},
		TraceID: block.TraceID,
	}
}

// ============================================================================
// MCP API 处理器
// ============================================================================

// handleInitialize 处理 MCP 初始化请求
func (p *MCPServerPlugin) handleInitialize(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "opencat-mcp-server",
				"version": "1.0.0",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolsListAPI 处理 MCP 工具列表请求
func (p *MCPServerPlugin) handleToolsListAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取 router 中的所有函数
	functions := p.router.List("")

	// 转换为 MCP 工具格式
	tools := make([]map[string]interface{}, 0)
	for _, fn := range functions {
		// 只暴露特定命名空间的函数
		if fn.Namespace == "util" || fn.Namespace == "sys" || fn.Namespace == "fs" {
			tool := map[string]interface{}{
				"name":        fn.Name,
				"description": fn.Description,
			}

			// 添加输入模式（简化）
			tool["inputSchema"] = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"args": map[string]interface{}{
						"type":        "object",
						"description": "函数参数",
					},
				},
			}

			tools = append(tools, tool)
		}
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"result": map[string]interface{}{
			"tools": tools,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolsCall 处理 MCP 工具调用请求
func (p *MCPServerPlugin) handleToolsCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      int                    `json:"id"`
		Method  string                 `json:"method"`
		Params  map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		p.sendError(w, 1, -32600, "Parse error", nil)
		return
	}

	toolName, ok := request.Params["name"].(string)
	if !ok {
		p.sendError(w, request.ID, -32602, "Invalid params", "Missing or invalid tool name")
		return
	}

	args, _ := request.Params["arguments"].(map[string]interface{})

	// 创建 router 上下文
	routerCtx := router.NewContext("mcp", "mcp-client", p.router)
	block := router.NewDataBlock(toolName, args)

	// 调用函数
	result := p.router.Call(routerCtx, block)

	// 构建响应
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request.ID,
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("%v", result.Data),
				},
			},
			"isError": !result.Success,
		},
	}

	if !result.Success && result.Error != nil {
		response["result"].(map[string]interface{})["error"] = map[string]interface{}{
			"code":    result.Error.Code,
			"message": result.Error.Message,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEvents 处理 MCP 事件请求
func (p *MCPServerPlugin) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 支持 Server-Sent Events (SSE) 或 WebSocket
	// 这里简化处理，只返回成功响应
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"result": map[string]interface{}{
			"success": true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealthAPI 处理 MCP 健康检查
func (p *MCPServerPlugin) handleHealthAPI(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	response := map[string]interface{}{
		"status":  "healthy",
		"running": running,
		"port":    p.port,
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendError 发送 MCP 错误响应
func (p *MCPServerPlugin) sendError(w http.ResponseWriter, id, code int, message string, data interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if data != nil {
		response["error"].(map[string]interface{})["data"] = data
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200) // JSON-RPC 2.0 总是返回 200
	json.NewEncoder(w).Encode(response)
}
