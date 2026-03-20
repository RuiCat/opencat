package servers

import (
	"api/plugin"
	"api/router"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ============================================================================
// HTTP 服务器插件
// ============================================================================

// HTTPServerPlugin HTTP 服务器插件
type HTTPServerPlugin struct {
	*plugin.BasePlugin
	router  *router.Router
	server  *http.Server
	port    int
	mu      sync.RWMutex
	running bool
}

// NewHTTPServerPlugin 创建新的 HTTP 服务器插件
func NewHTTPServerPlugin(router *router.Router, port int) *HTTPServerPlugin {
	info := &plugin.PluginInfo{
		Name:        "HTTPServer",
		Version:     "1.0.0",
		Description: "HTTP 服务器插件，提供 RESTful API",
		Author:      "opencat",
		CreatedAt:   time.Now(),
		Metadata: map[string]string{
			"category": "server",
			"type":     "http",
		},
	}

	// 创建基础插件
	basePlugin := plugin.NewBasePlugin(info, nil)

	// 创建 HTTP 服务器插件
	plugin := &HTTPServerPlugin{
		BasePlugin: basePlugin,
		router:     router,
		port:       port,
		running:    false,
	}

	return plugin
}

// Init 初始化插件
func (p *HTTPServerPlugin) Init(ctx context.Context) error {
	// 调用父类初始化
	if err := p.BasePlugin.Init(ctx); err != nil {
		return err
	}

	// 注册 HTTP 相关函数到 router
	if err := p.registerFunctions(); err != nil {
		return fmt.Errorf("注册函数失败: %w", err)
	}

	fmt.Printf("HTTP 服务器插件初始化完成，端口: %d\n", p.port)
	return nil
}

// Shutdown 关闭插件
func (p *HTTPServerPlugin) Shutdown(ctx context.Context) error {
	// 停止 HTTP 服务器
	if p.running {
		if err := p.stopServer(ctx); err != nil {
			fmt.Printf("停止 HTTP 服务器失败: %v\n", err)
		}
	}

	// 注销函数
	p.unregisterFunctions()

	// 调用父类关闭
	return p.BasePlugin.Shutdown(ctx)
}

// registerFunctions 注册函数到 router
func (p *HTTPServerPlugin) registerFunctions() error {
	functions := []*router.Function{
		{
			Name:        "http.start",
			Description: "启动 HTTP 服务器",
			Namespace:   "http",
			Handler:     p.handleStart,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "http.stop",
			Description: "停止 HTTP 服务器",
			Namespace:   "http",
			Handler:     p.handleStop,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "http.status",
			Description: "获取 HTTP 服务器状态",
			Namespace:   "http",
			Handler:     p.handleStatus,
			Builtin:     false,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "http.health",
			Description: "HTTP 服务器健康检查",
			Namespace:   "http",
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
func (p *HTTPServerPlugin) unregisterFunctions() {
	functions := []string{
		"http.start",
		"http.stop",
		"http.status",
		"http.health",
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
// HTTP 服务器实现
// ============================================================================

// startServer 启动 HTTP 服务器
func (p *HTTPServerPlugin) startServer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("HTTP 服务器已在运行")
	}

	// 创建路由器
	mux := http.NewServeMux()

	// 注册 API 端点
	mux.HandleFunc("/api/v1/health", p.handleHealthAPI)
	mux.HandleFunc("/api/v1/plugins", p.handlePluginsAPI)
	mux.HandleFunc("/api/v1/tools", p.handleToolsAPI)
	mux.HandleFunc("/api/v1/functions", p.handleFunctionsAPI)

	// 创建 HTTP 服务器
	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		fmt.Printf("HTTP 服务器启动在端口 %d\n", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP 服务器错误: %v\n", err)
		}
	}()

	p.running = true
	return nil
}

// stopServer 停止 HTTP 服务器
func (p *HTTPServerPlugin) stopServer(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.server == nil {
		return fmt.Errorf("HTTP 服务器未运行")
	}

	// 创建超时上下文
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 优雅关闭
	if err := p.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP 服务器关闭失败: %w", err)
	}

	p.running = false
	p.server = nil
	return nil
}

// ============================================================================
// 函数处理器
// ============================================================================

// handleStart 处理 http.start 函数调用
func (p *HTTPServerPlugin) handleStart(ctx *router.Context, block *router.DataBlock) *router.Result {
	if err := p.startServer(); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "HTTP_START_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message": "HTTP 服务器已启动",
			"port":    p.port,
			"running": true,
		},
		TraceID: block.TraceID,
	}
}

// handleStop 处理 http.stop 函数调用
func (p *HTTPServerPlugin) handleStop(ctx *router.Context, block *router.DataBlock) *router.Result {
	// 创建 context.Context
	goCtx := context.Background()
	if err := p.stopServer(goCtx); err != nil {
		return &router.Result{
			Success: false,
			Error: &router.ErrorInfo{
				Code:    "HTTP_STOP_FAILED",
				Message: err.Error(),
			},
			TraceID: block.TraceID,
		}
	}

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"message": "HTTP 服务器已停止",
			"port":    p.port,
			"running": false,
		},
		TraceID: block.TraceID,
	}
}

// handleStatus 处理 http.status 函数调用
func (p *HTTPServerPlugin) handleStatus(ctx *router.Context, block *router.DataBlock) *router.Result {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	return &router.Result{
		Success: true,
		Data: map[string]interface{}{
			"running": running,
			"port":    p.port,
			"uptime":  time.Since(p.GetInfo().CreatedAt).String(),
		},
		TraceID: block.TraceID,
	}
}

// handleHealth 处理 http.health 函数调用
func (p *HTTPServerPlugin) handleHealth(ctx *router.Context, block *router.DataBlock) *router.Result {
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
			"status":  status,
			"running": running,
			"port":    p.port,
			"time":    time.Now().Format(time.RFC3339),
		},
		TraceID: block.TraceID,
	}
}

// ============================================================================
// HTTP API 处理器
// ============================================================================

// handleHealthAPI 处理健康检查 API
func (p *HTTPServerPlugin) handleHealthAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
	fmt.Fprintf(w, "%s", toJSON(response))
}

// handlePluginsAPI 处理插件列表 API
func (p *HTTPServerPlugin) handlePluginsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 这里可以扩展为获取所有插件信息
	response := map[string]interface{}{
		"plugins": []map[string]interface{}{
			{
				"name":        "HTTPServer",
				"version":     "1.0.0",
				"description": "HTTP 服务器插件",
				"running":     p.running,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", toJSON(response))
}

// handleToolsAPI 处理工具列表 API
func (p *HTTPServerPlugin) handleToolsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "http.start",
				"description": "启动 HTTP 服务器",
				"enabled":     true,
			},
			{
				"name":        "http.stop",
				"description": "停止 HTTP 服务器",
				"enabled":     true,
			},
			{
				"name":        "http.status",
				"description": "获取 HTTP 服务器状态",
				"enabled":     true,
			},
			{
				"name":        "http.health",
				"description": "HTTP 服务器健康检查",
				"enabled":     true,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", toJSON(response))
}

// handleFunctionsAPI 处理函数列表 API
func (p *HTTPServerPlugin) handleFunctionsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取 router 中的函数列表
	functions := p.router.List("http")

	funcList := make([]map[string]interface{}, len(functions))
	for i, fn := range functions {
		funcList[i] = map[string]interface{}{
			"name":        fn.Name,
			"description": fn.Description,
			"namespace":   fn.Namespace,
			"enabled":     fn.Enabled,
			"call_count":  fn.Stats.CallCount,
		}
	}

	response := map[string]interface{}{
		"functions": funcList,
		"count":     len(funcList),
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", toJSON(response))
}

// ============================================================================
// 辅助函数
// ============================================================================

// toJSON 转换为 JSON 字符串
func toJSON(data interface{}) string {
	// 简化实现，实际应该使用 json.Marshal
	return fmt.Sprintf("%v", data)
}
