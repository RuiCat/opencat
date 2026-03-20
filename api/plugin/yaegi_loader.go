package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ============================================================================
// Yaegi 插件加载器
// ============================================================================

// YaegiLoader Yaegi 插件加载器
type YaegiLoader struct {
	mu         sync.RWMutex
	plugins    map[string]*YaegiPlugin
	interpOpts interp.Options
	router     Router
}

// YaegiPlugin Yaegi 插件
type YaegiPlugin struct {
	info        *PluginInfo
	interpreter *interp.Interpreter
	enabled     bool
	tools       map[string]ToolHandler
	events      map[string][]EventHandler
}

// NewYaegiLoader 创建新的 Yaegi 加载器
func NewYaegiLoader(router Router) *YaegiLoader {
	return &YaegiLoader{
		plugins: make(map[string]*YaegiPlugin),
		interpOpts: interp.Options{
			GoPath: os.Getenv("GOPATH"),
		},
		router: router,
	}
}

// LoadPlugin 加载 Yaegi 插件
func (yl *YaegiLoader) LoadPlugin(name, scriptPath string) error {
	yl.mu.Lock()
	defer yl.mu.Unlock()

	if _, exists := yl.plugins[name]; exists {
		return ErrPluginAlreadyLoaded
	}

	// 读取脚本文件
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取脚本文件失败: %w", err)
	}

	// 创建解释器
	i := interp.New(yl.interpOpts)

	// 使用标准库
	if err := i.Use(stdlib.Symbols); err != nil {
		return fmt.Errorf("加载标准库失败: %w", err)
	}

	// 注册插件接口
	if err := yl.registerPluginAPI(i); err != nil {
		return fmt.Errorf("注册插件API失败: %w", err)
	}

	// 执行脚本
	_, err = i.Eval(string(scriptContent))
	if err != nil {
		return fmt.Errorf("执行脚本失败: %w", err)
	}

	// 获取插件信息
	pluginInfo, err := yl.extractPluginInfo(i)
	if err != nil {
		return fmt.Errorf("提取插件信息失败: %w", err)
	}

	// 创建插件实例
	plugin := &YaegiPlugin{
		info:        pluginInfo,
		interpreter: i,
		enabled:     true,
		tools:       make(map[string]ToolHandler),
		events:      make(map[string][]EventHandler),
	}

	// 初始化插件
	if err := yl.initPlugin(i, plugin); err != nil {
		return fmt.Errorf("初始化插件失败: %w", err)
	}

	yl.plugins[name] = plugin
	return nil
}

// registerPluginAPI 注册插件API到解释器
func (yl *YaegiLoader) registerPluginAPI(i *interp.Interpreter) error {
	// 创建函数调用器实例
	functionCaller := &routerFunctionCaller{router: yl.router}

	// 注册插件类型和接口
	exports := interp.Exports{
		"plugin/plugin": map[string]reflect.Value{
			// 类型
			"PluginInfo":     reflect.ValueOf((*PluginInfo)(nil)),
			"ToolDefinition": reflect.ValueOf((*ToolDefinition)(nil)),
			"ToolSchema":     reflect.ValueOf((*ToolSchema)(nil)),
			"ToolResult":     reflect.ValueOf((*ToolResult)(nil)),
			"Event":          reflect.ValueOf((*Event)(nil)),

			// 错误
			"ErrPluginNotFound":        reflect.ValueOf(ErrPluginNotFound),
			"ErrToolNotFound":          reflect.ValueOf(ErrToolNotFound),
			"ErrToolAlreadyRegistered": reflect.ValueOf(ErrToolAlreadyRegistered),

			// 函数调用接口
			"FunctionCaller": reflect.ValueOf((*FunctionCaller)(nil)),

			// 函数调用器实例
			"GetFunctionCaller": reflect.ValueOf(func() FunctionCaller {
				return functionCaller
			}),

			// Router 接口
			"Router": reflect.ValueOf((*Router)(nil)),
		},
	}

	return i.Use(exports)
}

// extractPluginInfo 从解释器提取插件信息
func (yl *YaegiLoader) extractPluginInfo(i *interp.Interpreter) (*PluginInfo, error) {
	// 获取 PluginInfo 变量
	v, err := i.Eval("PluginInfo")
	if err != nil {
		return nil, fmt.Errorf("PluginInfo 未定义: %w", err)
	}

	// 类型断言
	pluginInfo, ok := v.Interface().(*PluginInfo)
	if !ok {
		return nil, fmt.Errorf("PluginInfo 类型错误")
	}

	if pluginInfo.Name == "" {
		return nil, fmt.Errorf("插件名称不能为空")
	}

	return pluginInfo, nil
}

// initPlugin 初始化插件
func (yl *YaegiLoader) initPlugin(i *interp.Interpreter, plugin *YaegiPlugin) error {
	// 检查 PluginInit 函数是否存在
	v, err := i.Eval("PluginInit")
	if err != nil {
		// PluginInit 可选
		return nil
	}

	// 获取函数并调用
	fn, ok := v.Interface().(func(*yaegiPluginAPI) error)
	if !ok {
		return fmt.Errorf("PluginInit 签名错误，应该是 func(*yaegiPluginAPI) error")
	}

	// 创建插件API
	api := yl.createPluginAPI(plugin)
	return fn(api)
}

// createPluginAPI 创建插件API
func (yl *YaegiLoader) createPluginAPI(plugin *YaegiPlugin) *yaegiPluginAPI {
	return &yaegiPluginAPI{
		plugin: plugin,
		loader: yl,
		router: yl.router,
	}
}

// routerFunctionCaller 路由函数调用器实现
type routerFunctionCaller struct {
	router Router
}

// Call 调用函数
func (r *routerFunctionCaller) Call(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	if r.router == nil {
		return nil, &PluginError{Message: "路由未设置"}
	}
	return r.router.CallFunction(ctx, name, args)
}

// Has 检查函数是否存在
func (r *routerFunctionCaller) Has(name string) bool {
	if r.router == nil {
		return false
	}
	return r.router.HasFunction(name)
}

// List 列出所有可用函数
func (r *routerFunctionCaller) List() []string {
	if r.router == nil {
		return []string{}
	}
	return r.router.ListFunctions()
}

// GetInfo 获取函数信息
func (r *routerFunctionCaller) GetInfo(name string) (map[string]interface{}, error) {
	if r.router == nil {
		return nil, &PluginError{Message: "路由未设置"}
	}

	// 这里可以扩展以获取更多函数信息
	// 目前只返回基本信息和是否存在
	info := map[string]interface{}{
		"name":   name,
		"exists": r.router.HasFunction(name),
		"type":   "function",
		"source": "router",
	}

	return info, nil
}

// yaegiPluginAPI Yaegi 插件API实现
type yaegiPluginAPI struct {
	plugin *YaegiPlugin
	loader *YaegiLoader
	router Router
}

// GetInfo 获取插件信息
func (api *yaegiPluginAPI) GetInfo() *PluginInfo {
	return api.plugin.info
}

// Init 初始化插件
func (api *yaegiPluginAPI) Init(ctx context.Context) error {
	api.plugin.enabled = true
	return nil
}

// Shutdown 关闭插件
func (api *yaegiPluginAPI) Shutdown(ctx context.Context) error {
	api.plugin.enabled = false
	api.plugin.tools = make(map[string]ToolHandler)
	api.plugin.events = make(map[string][]EventHandler)
	return nil
}

// Enable 启用插件
func (api *yaegiPluginAPI) Enable() error {
	api.plugin.enabled = true
	return nil
}

// Disable 禁用插件
func (api *yaegiPluginAPI) Disable() error {
	api.plugin.enabled = false
	return nil
}

// IsEnabled 检查插件是否启用
func (api *yaegiPluginAPI) IsEnabled() bool {
	return api.plugin.enabled
}

// RegisterTool 注册工具
func (api *yaegiPluginAPI) RegisterTool(tool ToolDefinition, handler ToolHandler) error {
	api.plugin.tools[tool.Name] = handler
	return nil
}

// UnregisterTool 注销工具
func (api *yaegiPluginAPI) UnregisterTool(name string) error {
	delete(api.plugin.tools, name)
	return nil
}

// ListTools 列出工具
func (api *yaegiPluginAPI) ListTools() []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(api.plugin.tools))
	for name := range api.plugin.tools {
		tools = append(tools, ToolDefinition{
			Name:        name,
			Description: "Yaegi 插件工具",
			Enabled:     true,
		})
	}
	return tools
}

// CallTool 调用工具
func (api *yaegiPluginAPI) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	handler, exists := api.plugin.tools[name]
	if !exists {
		return nil, ErrToolNotFound
	}

	if !api.plugin.enabled {
		return &ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	return handler(ctx, args)
}

// EmitEvent 发送事件
func (api *yaegiPluginAPI) EmitEvent(ctx context.Context, event *Event) error {
	if api.router != nil {
		return api.router.PublishEvent(event.Type, event.Data)
	}
	return nil
}

// SubscribeEvent 订阅事件
func (api *yaegiPluginAPI) SubscribeEvent(eventType string, handler EventHandler) error {
	api.plugin.events[eventType] = append(api.plugin.events[eventType], handler)
	return nil
}

// UnsubscribeEvent 取消订阅事件
func (api *yaegiPluginAPI) UnsubscribeEvent(eventType string, handler EventHandler) error {
	handlers, exists := api.plugin.events[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if &h == &handler {
			api.plugin.events[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// LoadFromDirectory 从目录加载所有插件
func (yl *YaegiLoader) LoadFromDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只加载 .go 文件
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		pluginPath := filepath.Join(dirPath, entry.Name())
		pluginName := entry.Name()[:len(entry.Name())-3] // 去掉 .go 扩展名

		if err := yl.LoadPlugin(pluginName, pluginPath); err != nil {
			fmt.Printf("加载插件 %s 失败: %v\n", pluginName, err)
			continue
		}

		fmt.Printf("加载插件成功: %s\n", pluginName)
	}

	return nil
}

// GetPlugin 获取插件
func (yl *YaegiLoader) GetPlugin(name string) (Plugin, error) {
	yl.mu.RLock()
	defer yl.mu.RUnlock()

	plugin, exists := yl.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return &yaegiPluginWrapper{plugin: plugin, loader: yl, router: yl.router}, nil
}

// UnloadPlugin 卸载插件
func (yl *YaegiLoader) UnloadPlugin(name string) error {
	yl.mu.Lock()
	defer yl.mu.Unlock()

	plugin, exists := yl.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	// 关闭插件
	ctx := context.Background()
	api := yl.createPluginAPI(plugin)
	if err := api.Shutdown(ctx); err != nil {
		return fmt.Errorf("插件关闭失败: %w", err)
	}

	delete(yl.plugins, name)
	return nil
}

// ListPlugins 列出所有插件
func (yl *YaegiLoader) ListPlugins() []PluginInfo {
	yl.mu.RLock()
	defer yl.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(yl.plugins))
	for _, plugin := range yl.plugins {
		infos = append(infos, *plugin.info)
	}
	return infos
}

// yaegiPluginWrapper Yaegi 插件包装器
type yaegiPluginWrapper struct {
	plugin *YaegiPlugin
	loader *YaegiLoader
	router Router
}

func (w *yaegiPluginWrapper) GetInfo() *PluginInfo {
	return w.plugin.info
}

func (w *yaegiPluginWrapper) Init(ctx context.Context) error {
	w.plugin.enabled = true
	return nil
}

func (w *yaegiPluginWrapper) Shutdown(ctx context.Context) error {
	w.plugin.enabled = false
	w.plugin.tools = make(map[string]ToolHandler)
	w.plugin.events = make(map[string][]EventHandler)
	return nil
}

func (w *yaegiPluginWrapper) Enable() error {
	w.plugin.enabled = true
	return nil
}

func (w *yaegiPluginWrapper) Disable() error {
	w.plugin.enabled = false
	return nil
}

func (w *yaegiPluginWrapper) IsEnabled() bool {
	return w.plugin.enabled
}

func (w *yaegiPluginWrapper) RegisterTool(tool ToolDefinition, handler ToolHandler) error {
	w.plugin.tools[tool.Name] = handler
	return nil
}

func (w *yaegiPluginWrapper) UnregisterTool(name string) error {
	delete(w.plugin.tools, name)
	return nil
}

func (w *yaegiPluginWrapper) ListTools() []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(w.plugin.tools))
	for name := range w.plugin.tools {
		tools = append(tools, ToolDefinition{
			Name:        name,
			Description: "Yaegi 插件工具",
			Enabled:     true,
		})
	}
	return tools
}

func (w *yaegiPluginWrapper) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	handler, exists := w.plugin.tools[name]
	if !exists {
		return nil, ErrToolNotFound
	}

	if !w.plugin.enabled {
		return &ToolResult{
			Success: false,
			Error:   "插件已禁用",
		}, nil
	}

	return handler(ctx, args)
}

func (w *yaegiPluginWrapper) EmitEvent(ctx context.Context, event *Event) error {
	if w.router != nil {
		return w.router.PublishEvent(event.Type, event.Data)
	}
	return nil
}

func (w *yaegiPluginWrapper) SubscribeEvent(eventType string, handler EventHandler) error {
	w.plugin.events[eventType] = append(w.plugin.events[eventType], handler)
	return nil
}

func (w *yaegiPluginWrapper) UnsubscribeEvent(eventType string, handler EventHandler) error {
	handlers, exists := w.plugin.events[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if &h == &handler {
			w.plugin.events[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// CheckDependencies 检查插件依赖
func (w *yaegiPluginWrapper) CheckDependencies(ctx context.Context) error {
	// 获取插件依赖
	deps := w.GetDependencies()
	if len(deps) == 0 {
		return nil
	}

	// 检查路由中是否存在这些函数
	if w.router == nil {
		return fmt.Errorf("路由未设置，无法检查依赖")
	}

	for _, dep := range deps {
		// 检查函数是否存在
		if !w.router.HasFunction(dep) {
			return fmt.Errorf("依赖函数不存在: %s", dep)
		}
	}

	return nil
}

// GetDependencies 获取插件依赖
func (w *yaegiPluginWrapper) GetDependencies() []string {
	if w.plugin.info == nil {
		return []string{}
	}
	return w.plugin.info.Dependencies
}
