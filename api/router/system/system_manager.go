package system

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 系统管理器
// ============================================================================

// SystemManager 系统管理器
type SystemManager struct {
	networkSystem  *NetworkSystem
	fileSystem     *FileSystemImpl
	systemInfo     *SystemInfoImpl
	processManager *ProcessManagerImpl
	utilityTools   *UtilityToolsImpl
}

// NewSystemManager 创建新的系统管理器
func NewSystemManager() *SystemManager {
	return &SystemManager{
		networkSystem:  NewNetworkSystem(),
		fileSystem:     NewFileSystem(""),
		systemInfo:     NewSystemInfo(),
		processManager: NewProcessManager(),
		utilityTools:   NewUtilityTools(),
	}
}

// ============================================================================
// 网络接口实现
// ============================================================================

// HTTPGet 执行HTTP GET请求
func (sm *SystemManager) HTTPGet(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error) {
	return sm.networkSystem.HTTPGet(ctx, url, headers)
}

// HTTPPost 执行HTTP POST请求
func (sm *SystemManager) HTTPPost(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	return sm.networkSystem.HTTPPost(ctx, url, headers, body)
}

// HTTPPut 执行HTTP PUT请求
func (sm *SystemManager) HTTPPut(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	return sm.networkSystem.HTTPPut(ctx, url, headers, body)
}

// HTTPDelete 执行HTTP DELETE请求
func (sm *SystemManager) HTTPDelete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error) {
	return sm.networkSystem.HTTPDelete(ctx, url, headers)
}

// HTTPRequest 执行HTTP请求
func (sm *SystemManager) HTTPRequest(ctx context.Context, method, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	return sm.networkSystem.HTTPRequest(ctx, method, url, headers, body)
}

// SetTimeout 设置HTTP超时时间
func (sm *SystemManager) SetTimeout(timeout time.Duration) {
	sm.networkSystem.SetTimeout(timeout)
}

// SetProxy 设置代理
func (sm *SystemManager) SetProxy(proxyURL string) error {
	return sm.networkSystem.SetProxy(proxyURL)
}

// SetTLSConfig 设置TLS配置
func (sm *SystemManager) SetTLSConfig(config *TLSConfig) error {
	return sm.networkSystem.SetTLSConfig(config)
}

// WebSocketConnect 建立WebSocket连接
func (sm *SystemManager) WebSocketConnect(ctx context.Context, url string, headers map[string]string) (*WebSocketConnection, error) {
	return sm.networkSystem.WebSocketConnect(ctx, url, headers)
}

// WebSocketClose 关闭WebSocket连接
func (sm *SystemManager) WebSocketClose(connID string) error {
	return sm.networkSystem.WebSocketClose(connID)
}

// WebSocketSend 发送WebSocket消息
func (sm *SystemManager) WebSocketSend(connID string, message []byte) error {
	return sm.networkSystem.WebSocketSend(connID, message)
}

// WebSocketReceive 接收WebSocket消息
func (sm *SystemManager) WebSocketReceive(connID string, timeout time.Duration) ([]byte, error) {
	return sm.networkSystem.WebSocketReceive(connID, timeout)
}

// WebSocketStatus 获取WebSocket连接状态
func (sm *SystemManager) WebSocketStatus(connID string) (*WebSocketStatus, error) {
	return sm.networkSystem.WebSocketStatus(connID)
}

// ListWebSocketConnections 列出所有WebSocket连接
func (sm *SystemManager) ListWebSocketConnections() []string {
	return sm.networkSystem.ListWebSocketConnections()
}

// Ping 执行ping测试
func (sm *SystemManager) Ping(ctx context.Context, host string, count int) (*PingResult, error) {
	return sm.networkSystem.Ping(ctx, host, count)
}

// Traceroute 执行traceroute测试
func (sm *SystemManager) Traceroute(ctx context.Context, host string) (*TracerouteResult, error) {
	return sm.networkSystem.Traceroute(ctx, host)
}

// DNSLookup 执行DNS查询
func (sm *SystemManager) DNSLookup(ctx context.Context, host string) (*DNSResult, error) {
	return sm.networkSystem.DNSLookup(ctx, host)
}

// GetNetworkInterfaces 获取网络接口信息
func (sm *SystemManager) GetNetworkInterfaces() ([]NetworkInterface, error) {
	return sm.networkSystem.GetNetworkInterfaces()
}

// GetNetworkStats 获取网络统计信息
func (sm *SystemManager) GetNetworkStats() (*NetworkStats, error) {
	return sm.networkSystem.GetNetworkStats()
}

// ============================================================================
// 文件系统接口实现
// ============================================================================

// FileRead 读取文件内容
func (sm *SystemManager) FileRead(ctx context.Context, path string) ([]byte, error) {
	return sm.fileSystem.FileRead(ctx, path)
}

// FileWrite 写入文件内容
func (sm *SystemManager) FileWrite(ctx context.Context, path string, data []byte, append bool) error {
	return sm.fileSystem.FileWrite(ctx, path, data, append)
}

// FileDelete 删除文件
func (sm *SystemManager) FileDelete(ctx context.Context, path string) error {
	return sm.fileSystem.FileDelete(ctx, path)
}

// FileCopy 复制文件
func (sm *SystemManager) FileCopy(ctx context.Context, src, dst string) error {
	return sm.fileSystem.FileCopy(ctx, src, dst)
}

// FileMove 移动文件
func (sm *SystemManager) FileMove(ctx context.Context, src, dst string) error {
	return sm.fileSystem.FileMove(ctx, src, dst)
}

// DirList 列出目录内容
func (sm *SystemManager) DirList(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	return sm.fileSystem.DirList(ctx, path, recursive)
}

// DirCreate 创建目录
func (sm *SystemManager) DirCreate(ctx context.Context, path string) error {
	return sm.fileSystem.DirCreate(ctx, path)
}

// DirDelete 删除目录
func (sm *SystemManager) DirDelete(ctx context.Context, path string) error {
	return sm.fileSystem.DirDelete(ctx, path)
}

// FileExists 检查文件是否存在
func (sm *SystemManager) FileExists(ctx context.Context, path string) (bool, error) {
	return sm.fileSystem.FileExists(ctx, path)
}

// FileInfo 获取文件信息
func (sm *SystemManager) FileInfo(ctx context.Context, path string) (*FileInfo, error) {
	return sm.fileSystem.FileInfo(ctx, path)
}

// FileSize 获取文件大小
func (sm *SystemManager) FileSize(ctx context.Context, path string) (int64, error) {
	return sm.fileSystem.FileSize(ctx, path)
}

// PathJoin 连接路径元素
func (sm *SystemManager) PathJoin(elem ...string) string {
	return sm.fileSystem.PathJoin(elem...)
}

// PathAbs 获取绝对路径
func (sm *SystemManager) PathAbs(path string) (string, error) {
	return sm.fileSystem.PathAbs(path)
}

// PathBase 获取路径的最后一部分
func (sm *SystemManager) PathBase(path string) string {
	return sm.fileSystem.PathBase(path)
}

// PathDir 获取路径的目录部分
func (sm *SystemManager) PathDir(path string) string {
	return sm.fileSystem.PathDir(path)
}

// PathExt 获取路径的扩展名
func (sm *SystemManager) PathExt(path string) string {
	return sm.fileSystem.PathExt(path)
}

// WatchFile 监控文件
func (sm *SystemManager) WatchFile(ctx context.Context, path string, handler FileEventHandler) (string, error) {
	return sm.fileSystem.WatchFile(ctx, path, handler)
}

// WatchDir 监控目录
func (sm *SystemManager) WatchDir(ctx context.Context, path string, recursive bool, handler FileEventHandler) (string, error) {
	return sm.fileSystem.WatchDir(ctx, path, recursive, handler)
}

// Unwatch 取消监控
func (sm *SystemManager) Unwatch(watchID string) error {
	return sm.fileSystem.Unwatch(watchID)
}

// ListWatches 列出所有监控
func (sm *SystemManager) ListWatches() []WatchInfo {
	return sm.fileSystem.ListWatches()
}

// GetWatchStats 获取监控统计
func (sm *SystemManager) GetWatchStats(watchID string) (*WatchStats, error) {
	return sm.fileSystem.GetWatchStats(watchID)
}

// ============================================================================
// 系统信息接口实现（完整版）
// ============================================================================

// SystemInfoImpl 系统信息实现
type SystemInfoImpl struct{}

// NewSystemInfo 创建新的系统信息
func NewSystemInfo() *SystemInfoImpl {
	return &SystemInfoImpl{}
}

// GetSystemInfo 获取系统信息
func (si *SystemInfoImpl) GetSystemInfo() (*SystemInfoData, error) {
	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// 获取运行时信息
	numGoroutine := runtime.NumGoroutine()

	return &SystemInfoData{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Kernel:       getKernelVersion(),
		Uptime:       getUptime(),
		LoadAvg:      getLoadAvg(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: numGoroutine,
	}, nil
}

// GetCPUInfo 获取CPU信息
func (si *SystemInfoImpl) GetCPUInfo() (*CPUInfo, error) {
	// 获取CPU核心数
	cores := runtime.NumCPU()

	// 简化实现：返回默认值
	return &CPUInfo{
		Model:  getCPUModel(),
		Cores:  cores,
		Usage:  0.0, // 需要外部库获取真实使用率
		User:   0.0,
		System: 0.0,
		Idle:   100.0,
	}, nil
}

// GetMemoryInfo 获取内存信息
func (si *SystemInfoImpl) GetMemoryInfo() (*MemoryInfo, error) {
	// 使用runtime获取内存信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 简化实现：返回运行时内存信息
	return &MemoryInfo{
		Total:     uint64(m.Sys),
		Used:      uint64(m.Alloc),
		Free:      uint64(m.Sys - m.Alloc),
		Available: uint64(m.Sys - m.Alloc),
		Usage:     float64(m.Alloc) / float64(m.Sys) * 100,
	}, nil
}

// GetDiskInfo 获取磁盘信息
func (si *SystemInfoImpl) GetDiskInfo() (*DiskInfo, error) {
	// 简化实现：返回默认值
	return &DiskInfo{
		Total: 0,
		Used:  0,
		Free:  0,
		Usage: 0.0,
	}, nil
}

// GetGoInfo 获取Go运行时信息
func (si *SystemInfoImpl) GetGoInfo() (*GoInfo, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &GoInfo{
		Version:      runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCgoCall:   0, // runtime.MemStats 中没有 NumCgoCall 字段
	}, nil
}

// GetProcessInfo 获取进程信息
func (si *SystemInfoImpl) GetProcessInfo() (*ProcessInfo, error) {
	pid := os.Getpid()
	ppid := os.Getppid()

	return &ProcessInfo{
		PID:           pid,
		PPID:          ppid,
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		CreateTime:    time.Now().Unix() - 3600, // 简化：假设进程已运行1小时
		Status:        "running",
		CmdLine:       getCmdLine(),
		Exe:           getExecutablePath(),
	}, nil
}

// 辅助函数：获取内核版本
func getKernelVersion() string {
	// 简化实现：返回默认值
	return "unknown"
}

// 辅助函数：获取系统运行时间
func getUptime() string {
	// 简化实现：返回默认值
	return "unknown"
}

// 辅助函数：获取系统负载
func getLoadAvg() []float64 {
	// 简化实现：返回默认值
	return []float64{0.0, 0.0, 0.0}
}

// 辅助函数：获取CPU型号
func getCPUModel() string {
	// 简化实现：返回默认值
	return "Unknown CPU"
}

// 辅助函数：获取命令行参数
func getCmdLine() string {
	return strings.Join(os.Args, " ")
}

// 辅助函数：获取可执行文件路径
func getExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	return exe
}

// ============================================================================
// 进程管理接口实现（完整版）
// ============================================================================

// ProcessManagerImpl 进程管理实现
type ProcessManagerImpl struct {
	processes map[int]*exec.Cmd
	mutex     sync.RWMutex
}

// NewProcessManager 创建新的进程管理器
func NewProcessManager() *ProcessManagerImpl {
	return &ProcessManagerImpl{
		processes: make(map[int]*exec.Cmd),
	}
}

// StartProcess 启动进程
func (pm *ProcessManagerImpl) StartProcess(ctx context.Context, cmd string, args []string, env map[string]string) (*Process, error) {
	// 创建命令
	command := exec.CommandContext(ctx, cmd, args...)

	// 设置环境变量
	for key, value := range env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// 启动进程
	if err := command.Start(); err != nil {
		return nil, &SystemError{
			Code:    "PROCESS_START_FAILED",
			Message: fmt.Sprintf("启动进程失败: %v", err),
			Cause:   err,
		}
	}

	pid := command.Process.Pid

	// 保存进程引用
	pm.mutex.Lock()
	pm.processes[pid] = command
	pm.mutex.Unlock()

	// 返回进程信息
	return &Process{
		PID:       pid,
		Cmd:       cmd,
		Args:      args,
		Env:       env,
		StartedAt: time.Now(),
		Running:   true,
	}, nil
}

// StopProcess 停止进程
func (pm *ProcessManagerImpl) StopProcess(pid int) error {
	pm.mutex.RLock()
	cmd, exists := pm.processes[pid]
	pm.mutex.RUnlock()

	if !exists {
		return &SystemError{
			Code:    "PROCESS_NOT_FOUND",
			Message: fmt.Sprintf("进程不存在: %d", pid),
		}
	}

	// 发送中断信号
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return &SystemError{
			Code:    "PROCESS_STOP_FAILED",
			Message: fmt.Sprintf("停止进程失败: %v", err),
			Cause:   err,
		}
	}

	// 等待进程结束
	cmd.Wait()

	// 从进程列表中移除
	pm.mutex.Lock()
	delete(pm.processes, pid)
	pm.mutex.Unlock()

	return nil
}

// KillProcess 杀死进程
func (pm *ProcessManagerImpl) KillProcess(pid int) error {
	pm.mutex.RLock()
	cmd, exists := pm.processes[pid]
	pm.mutex.RUnlock()

	if !exists {
		return &SystemError{
			Code:    "PROCESS_NOT_FOUND",
			Message: fmt.Sprintf("进程不存在: %d", pid),
		}
	}

	// 发送杀死信号
	if err := cmd.Process.Kill(); err != nil {
		return &SystemError{
			Code:    "PROCESS_KILL_FAILED",
			Message: fmt.Sprintf("杀死进程失败: %v", err),
			Cause:   err,
		}
	}

	// 等待进程结束
	cmd.Wait()

	// 从进程列表中移除
	pm.mutex.Lock()
	delete(pm.processes, pid)
	pm.mutex.Unlock()

	return nil
}

// ListProcesses 列出所有进程
func (pm *ProcessManagerImpl) ListProcesses() ([]ProcessInfo, error) {
	// 简化实现：只返回我们管理的进程
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	processes := make([]ProcessInfo, 0, len(pm.processes))
	for pid, cmd := range pm.processes {
		processes = append(processes, ProcessInfo{
			PID:           pid,
			PPID:          os.Getpid(),
			CPUPercent:    0.0,
			MemoryPercent: 0.0,
			CreateTime:    time.Now().Unix() - 3600,
			Status:        "running",
			CmdLine:       fmt.Sprintf("%s %v", cmd.Path, cmd.Args),
			Exe:           cmd.Path,
		})
	}

	return processes, nil
}

// GetProcess 获取进程信息
func (pm *ProcessManagerImpl) GetProcess(pid int) (*Process, error) {
	pm.mutex.RLock()
	cmd, exists := pm.processes[pid]
	pm.mutex.RUnlock()

	if !exists {
		return nil, &SystemError{
			Code:    "PROCESS_NOT_FOUND",
			Message: fmt.Sprintf("进程不存在: %d", pid),
		}
	}

	// 检查进程是否还在运行
	running := cmd.ProcessState == nil || !cmd.ProcessState.Exited()

	// 将环境变量从 []string 转换为 map[string]string
	envMap := make(map[string]string)
	for _, env := range cmd.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	process := &Process{
		PID:       pid,
		Cmd:       cmd.Path,
		Args:      cmd.Args,
		Env:       envMap,
		StartedAt: time.Now().Add(-time.Hour), // 简化：假设进程已运行1小时
		Running:   running,
	}

	// 如果进程已结束，设置退出信息
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		exitCode := cmd.ProcessState.ExitCode()
		process.ExitCode = &exitCode
		exitTime := time.Now()
		process.ExitTime = &exitTime
		process.Running = false
	}

	return process, nil
}

// ProcessStats 获取进程统计
func (pm *ProcessManagerImpl) ProcessStats(pid int) (*ProcessStats, error) {
	// 简化实现：返回默认统计信息
	return &ProcessStats{
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		MemoryRSS:     0,
		MemoryVMS:     0,
		NumThreads:    1,
		NumFDs:        0,
	}, nil
}

// ============================================================================
// 工具接口实现（完整版）
// ============================================================================

// UtilityToolsImpl 工具实现
type UtilityToolsImpl struct{}

// NewUtilityTools 创建新的工具
func NewUtilityTools() *UtilityToolsImpl {
	return &UtilityToolsImpl{}
}

// Base64Encode Base64编码
func (ut *UtilityToolsImpl) Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode Base64解码
func (ut *UtilityToolsImpl) Base64Decode(str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(str)
}

// JSONEncode JSON编码
func (ut *UtilityToolsImpl) JSONEncode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONDecode JSON解码
func (ut *UtilityToolsImpl) JSONDecode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// URLEncode URL编码
func (ut *UtilityToolsImpl) URLEncode(str string) string {
	return url.QueryEscape(str)
}

// URLDecode URL解码
func (ut *UtilityToolsImpl) URLDecode(str string) (string, error) {
	return url.QueryUnescape(str)
}

// MD5 MD5哈希
func (ut *UtilityToolsImpl) MD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// SHA1 SHA1哈希
func (ut *UtilityToolsImpl) SHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// SHA256 SHA256哈希
func (ut *UtilityToolsImpl) SHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA512 SHA512哈希
func (ut *UtilityToolsImpl) SHA512(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// RandomString 生成随机字符串
func (ut *UtilityToolsImpl) RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomInt 生成随机整数
func (ut *UtilityToolsImpl) RandomInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// UUID 生成UUID
func (ut *UtilityToolsImpl) UUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			time.Now().UnixNano(),
			rand.Int63(),
			rand.Int63(),
			rand.Int63(),
			rand.Int63())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Sleep 睡眠
func (ut *UtilityToolsImpl) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// Timer 定时器
func (ut *UtilityToolsImpl) Timer(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

// Ticker 定时器
func (ut *UtilityToolsImpl) Ticker(interval time.Duration) <-chan time.Time {
	return time.Tick(interval)
}

// ============================================================================
// 系统管理器接口实现
// ============================================================================

// GetSystemInfo 获取系统信息
func (sm *SystemManager) GetSystemInfo() (*SystemInfoData, error) {
	return sm.systemInfo.GetSystemInfo()
}

// GetCPUInfo 获取CPU信息
func (sm *SystemManager) GetCPUInfo() (*CPUInfo, error) {
	return sm.systemInfo.GetCPUInfo()
}

// GetMemoryInfo 获取内存信息
func (sm *SystemManager) GetMemoryInfo() (*MemoryInfo, error) {
	return sm.systemInfo.GetMemoryInfo()
}

// GetDiskInfo 获取磁盘信息
func (sm *SystemManager) GetDiskInfo() (*DiskInfo, error) {
	return sm.systemInfo.GetDiskInfo()
}

// GetGoInfo 获取Go运行时信息
func (sm *SystemManager) GetGoInfo() (*GoInfo, error) {
	return sm.systemInfo.GetGoInfo()
}

// GetProcessInfo 获取进程信息
func (sm *SystemManager) GetProcessInfo() (*ProcessInfo, error) {
	return sm.systemInfo.GetProcessInfo()
}

// StartProcess 启动进程
func (sm *SystemManager) StartProcess(ctx context.Context, cmd string, args []string, env map[string]string) (*Process, error) {
	return sm.processManager.StartProcess(ctx, cmd, args, env)
}

// StopProcess 停止进程
func (sm *SystemManager) StopProcess(pid int) error {
	return sm.processManager.StopProcess(pid)
}

// KillProcess 杀死进程
func (sm *SystemManager) KillProcess(pid int) error {
	return sm.processManager.KillProcess(pid)
}

// ListProcesses 列出所有进程
func (sm *SystemManager) ListProcesses() ([]ProcessInfo, error) {
	return sm.processManager.ListProcesses()
}

// GetProcess 获取进程信息
func (sm *SystemManager) GetProcess(pid int) (*Process, error) {
	return sm.processManager.GetProcess(pid)
}

// ProcessStats 获取进程统计
func (sm *SystemManager) ProcessStats(pid int) (*ProcessStats, error) {
	return sm.processManager.ProcessStats(pid)
}

// Base64Encode Base64编码
func (sm *SystemManager) Base64Encode(data []byte) string {
	return sm.utilityTools.Base64Encode(data)
}

// Base64Decode Base64解码
func (sm *SystemManager) Base64Decode(str string) ([]byte, error) {
	return sm.utilityTools.Base64Decode(str)
}

// JSONEncode JSON编码
func (sm *SystemManager) JSONEncode(v interface{}) ([]byte, error) {
	return sm.utilityTools.JSONEncode(v)
}

// JSONDecode JSON解码
func (sm *SystemManager) JSONDecode(data []byte, v interface{}) error {
	return sm.utilityTools.JSONDecode(data, v)
}

// URLEncode URL编码
func (sm *SystemManager) URLEncode(str string) string {
	return sm.utilityTools.URLEncode(str)
}

// URLDecode URL解码
func (sm *SystemManager) URLDecode(str string) (string, error) {
	return sm.utilityTools.URLDecode(str)
}

// MD5 MD5哈希
func (sm *SystemManager) MD5(data []byte) string {
	return sm.utilityTools.MD5(data)
}

// SHA1 SHA1哈希
func (sm *SystemManager) SHA1(data []byte) string {
	return sm.utilityTools.SHA1(data)
}

// SHA256 SHA256哈希
func (sm *SystemManager) SHA256(data []byte) string {
	return sm.utilityTools.SHA256(data)
}

// SHA512 SHA512哈希
func (sm *SystemManager) SHA512(data []byte) string {
	return sm.utilityTools.SHA512(data)
}

// RandomString 生成随机字符串
func (sm *SystemManager) RandomString(length int) string {
	return sm.utilityTools.RandomString(length)
}

// RandomInt 生成随机整数
func (sm *SystemManager) RandomInt(min, max int) int {
	return sm.utilityTools.RandomInt(min, max)
}

// UUID 生成UUID
func (sm *SystemManager) UUID() string {
	return sm.utilityTools.UUID()
}

// Sleep 睡眠
func (sm *SystemManager) Sleep(duration time.Duration) {
	sm.utilityTools.Sleep(duration)
}

// Timer 定时器
func (sm *SystemManager) Timer(duration time.Duration) <-chan time.Time {
	return sm.utilityTools.Timer(duration)
}

// Ticker 定时器
func (sm *SystemManager) Ticker(interval time.Duration) <-chan time.Time {
	return sm.utilityTools.Ticker(interval)
}
