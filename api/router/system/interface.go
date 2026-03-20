package system

import (
	"context"
	"time"
)

// ============================================================================
// 基础系统接口定义
// ============================================================================

// SystemInterface 基础系统接口
type SystemInterface interface {
	// 网络接口
	HTTPClient
	WebSocketClient
	NetworkMonitor

	// 文件接口
	FileSystem
	FileWatcher

	// 系统接口
	SystemInfo
	ProcessManager

	// 工具接口
	UtilityTools
}

// ============================================================================
// HTTP客户端接口
// ============================================================================

// HTTPClient HTTP客户端接口
type HTTPClient interface {
	// HTTP请求
	HTTPGet(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)
	HTTPPost(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error)
	HTTPPut(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error)
	HTTPDelete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)
	HTTPRequest(ctx context.Context, method, url string, headers map[string]string, body []byte) (*HTTPResponse, error)

	// 客户端配置
	SetTimeout(timeout time.Duration)
	SetProxy(proxyURL string) error
	SetTLSConfig(config *TLSConfig) error
}

// HTTPResponse HTTP响应
type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	Duration   time.Duration       `json:"duration"`
	Error      string              `json:"error,omitempty"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
}

// ============================================================================
// WebSocket客户端接口
// ============================================================================

// WebSocketClient WebSocket客户端接口
type WebSocketClient interface {
	// 连接管理
	WebSocketConnect(ctx context.Context, url string, headers map[string]string) (*WebSocketConnection, error)
	WebSocketClose(connID string) error

	// 消息收发
	WebSocketSend(connID string, message []byte) error
	WebSocketReceive(connID string, timeout time.Duration) ([]byte, error)

	// 连接状态
	WebSocketStatus(connID string) (*WebSocketStatus, error)
	ListWebSocketConnections() []string
}

// WebSocketConnection WebSocket连接
type WebSocketConnection struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Connected bool      `json:"connected"`
	CreatedAt time.Time `json:"created_at"`
	LastPing  time.Time `json:"last_ping,omitempty"`
}

// WebSocketStatus WebSocket状态
type WebSocketStatus struct {
	Connected    bool      `json:"connected"`
	MessagesSent int       `json:"messages_sent"`
	MessagesRecv int       `json:"messages_received"`
	LastActivity time.Time `json:"last_activity"`
	ErrorCount   int       `json:"error_count"`
}

// ============================================================================
// 网络监控接口
// ============================================================================

// NetworkMonitor 网络监控接口
type NetworkMonitor interface {
	// 网络诊断
	Ping(ctx context.Context, host string, count int) (*PingResult, error)
	Traceroute(ctx context.Context, host string) (*TracerouteResult, error)
	DNSLookup(ctx context.Context, host string) (*DNSResult, error)

	// 网络信息
	GetNetworkInterfaces() ([]NetworkInterface, error)
	GetNetworkStats() (*NetworkStats, error)
}

// PingResult Ping结果
type PingResult struct {
	Host        string        `json:"host"`
	PacketsSent int           `json:"packets_sent"`
	PacketsRecv int           `json:"packets_received"`
	PacketLoss  float64       `json:"packet_loss"`
	MinRTT      time.Duration `json:"min_rtt"`
	MaxRTT      time.Duration `json:"max_rtt"`
	AvgRTT      time.Duration `json:"avg_rtt"`
	StdDevRTT   time.Duration `json:"stddev_rtt"`
}

// TracerouteResult Traceroute结果
type TracerouteResult struct {
	Host    string          `json:"host"`
	Hops    []TracerouteHop `json:"hops"`
	Success bool            `json:"success"`
}

// TracerouteHop Traceroute跳点
type TracerouteHop struct {
	Hop     int           `json:"hop"`
	Address string        `json:"address"`
	RTT     time.Duration `json:"rtt"`
}

// DNSResult DNS查询结果
type DNSResult struct {
	Host    string   `json:"host"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl,omitempty"`
}

// NetworkInterface 网络接口
type NetworkInterface struct {
	Name         string   `json:"name"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_addr"`
	Flags        []string `json:"flags"`
	Addrs        []string `json:"addrs"`
}

// NetworkStats 网络统计
type NetworkStats struct {
	Interfaces  int `json:"interfaces"`
	BytesSent   int `json:"bytes_sent"`
	BytesRecv   int `json:"bytes_received"`
	PacketsSent int `json:"packets_sent"`
	PacketsRecv int `json:"packets_received"`
	ErrorsIn    int `json:"errors_in"`
	ErrorsOut   int `json:"errors_out"`
	DropIn      int `json:"drop_in"`
	DropOut     int `json:"drop_out"`
}

// ============================================================================
// 文件系统接口
// ============================================================================

// FileSystem 文件系统接口
type FileSystem interface {
	// 文件操作
	FileRead(ctx context.Context, path string) ([]byte, error)
	FileWrite(ctx context.Context, path string, data []byte, append bool) error
	FileDelete(ctx context.Context, path string) error
	FileCopy(ctx context.Context, src, dst string) error
	FileMove(ctx context.Context, src, dst string) error

	// 目录操作
	DirList(ctx context.Context, path string, recursive bool) ([]FileInfo, error)
	DirCreate(ctx context.Context, path string) error
	DirDelete(ctx context.Context, path string) error

	// 文件信息
	FileExists(ctx context.Context, path string) (bool, error)
	FileInfo(ctx context.Context, path string) (*FileInfo, error)
	FileSize(ctx context.Context, path string) (int64, error)

	// 路径操作
	PathJoin(elem ...string) string
	PathAbs(path string) (string, error)
	PathBase(path string) string
	PathDir(path string) string
	PathExt(path string) string
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

// ============================================================================
// 文件监控接口
// ============================================================================

// FileWatcher 文件监控接口
type FileWatcher interface {
	// 监控管理
	WatchFile(ctx context.Context, path string, handler FileEventHandler) (string, error)
	WatchDir(ctx context.Context, path string, recursive bool, handler FileEventHandler) (string, error)
	Unwatch(watchID string) error

	// 监控状态
	ListWatches() []WatchInfo
	GetWatchStats(watchID string) (*WatchStats, error)
}

// FileEvent 文件事件
type FileEvent struct {
	Type      FileEventType `json:"type"`
	Path      string        `json:"path"`
	Timestamp time.Time     `json:"timestamp"`
	OldPath   string        `json:"old_path,omitempty"`
	Size      int64         `json:"size,omitempty"`
}

// FileEventType 文件事件类型
type FileEventType string

const (
	FileEventCreate FileEventType = "create"
	FileEventWrite  FileEventType = "write"
	FileEventRemove FileEventType = "remove"
	FileEventRename FileEventType = "rename"
	FileEventChmod  FileEventType = "chmod"
)

// FileEventHandler 文件事件处理器
type FileEventHandler func(event *FileEvent)

// WatchInfo 监控信息
type WatchInfo struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Type      string    `json:"type"` // "file" or "dir"
	Recursive bool      `json:"recursive"`
	CreatedAt time.Time `json:"created_at"`
	Events    int       `json:"events"`
}

// WatchStats 监控统计
type WatchStats struct {
	EventsTotal   int       `json:"events_total"`
	EventsLastMin int       `json:"events_last_min"`
	LastEvent     time.Time `json:"last_event"`
	ErrorCount    int       `json:"error_count"`
}

// ============================================================================
// 系统信息接口
// ============================================================================

// SystemInfo 系统信息接口
type SystemInfo interface {
	// 系统信息
	GetSystemInfo() (*SystemInfoData, error)
	GetCPUInfo() (*CPUInfo, error)
	GetMemoryInfo() (*MemoryInfo, error)
	GetDiskInfo() (*DiskInfo, error)

	// 运行时信息
	GetGoInfo() (*GoInfo, error)
	GetProcessInfo() (*ProcessInfo, error)
}

// SystemInfoData 系统信息
type SystemInfoData struct {
	Hostname     string    `json:"hostname"`
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	Kernel       string    `json:"kernel"`
	Uptime       string    `json:"uptime"`
	LoadAvg      []float64 `json:"load_avg"`
	NumCPU       int       `json:"num_cpu"`
	NumGoroutine int       `json:"num_goroutine"`
}

// CPUInfo CPU信息
type CPUInfo struct {
	Model       string  `json:"model"`
	Cores       int     `json:"cores"`
	Usage       float64 `json:"usage"`
	User        float64 `json:"user"`
	System      float64 `json:"system"`
	Idle        float64 `json:"idle"`
	Temperature float64 `json:"temperature,omitempty"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

// DiskInfo 磁盘信息
type DiskInfo struct {
	Total uint64  `json:"total"`
	Used  uint64  `json:"used"`
	Free  uint64  `json:"free"`
	Usage float64 `json:"usage"`
}

// GoInfo Go运行时信息
type GoInfo struct {
	Version      string `json:"version"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCgoCall   int64  `json:"num_cgo_call"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int     `json:"pid"`
	PPID          int     `json:"ppid"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	CreateTime    int64   `json:"create_time"`
	Status        string  `json:"status"`
	CmdLine       string  `json:"cmd_line"`
	Exe           string  `json:"exe"`
}

// ============================================================================
// 进程管理接口
// ============================================================================

// ProcessManager 进程管理接口
type ProcessManager interface {
	// 进程管理
	StartProcess(ctx context.Context, cmd string, args []string, env map[string]string) (*Process, error)
	StopProcess(pid int) error
	KillProcess(pid int) error
	ListProcesses() ([]ProcessInfo, error)

	// 进程状态
	GetProcess(pid int) (*Process, error)
	ProcessStats(pid int) (*ProcessStats, error)
}

// Process 进程
type Process struct {
	PID       int               `json:"pid"`
	Cmd       string            `json:"cmd"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	StartedAt time.Time         `json:"started_at"`
	ExitCode  *int              `json:"exit_code,omitempty"`
	ExitTime  *time.Time        `json:"exit_time,omitempty"`
	Running   bool              `json:"running"`
	Stdout    string            `json:"stdout,omitempty"`
	Stderr    string            `json:"stderr,omitempty"`
}

// ProcessStats 进程统计
type ProcessStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryRSS     uint64  `json:"memory_rss"`
	MemoryVMS     uint64  `json:"memory_vms"`
	NumThreads    int     `json:"num_threads"`
	NumFDs        int     `json:"num_fds"`
}

// ============================================================================
// 工具接口
// ============================================================================

// UtilityTools 工具接口
type UtilityTools interface {
	// 编码解码
	Base64Encode(data []byte) string
	Base64Decode(str string) ([]byte, error)
	JSONEncode(v interface{}) ([]byte, error)
	JSONDecode(data []byte, v interface{}) error
	URLEncode(str string) string
	URLDecode(str string) (string, error)

	// 哈希计算
	MD5(data []byte) string
	SHA1(data []byte) string
	SHA256(data []byte) string
	SHA512(data []byte) string

	// 随机生成
	RandomString(length int) string
	RandomInt(min, max int) int
	UUID() string

	// 时间工具
	Sleep(duration time.Duration)
	Timer(duration time.Duration) <-chan time.Time
	Ticker(interval time.Duration) <-chan time.Time
}

// ============================================================================
// 错误类型定义
// ============================================================================

// SystemError 系统错误
type SystemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

func (e *SystemError) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + " - " + e.Cause.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *SystemError) Unwrap() error {
	return e.Cause
}

// 预定义错误
var (
	ErrNotImplemented   = &SystemError{Code: "NOT_IMPLEMENTED", Message: "功能未实现"}
	ErrInvalidParameter = &SystemError{Code: "INVALID_PARAMETER", Message: "参数无效"}
	ErrFileNotFound     = &SystemError{Code: "FILE_NOT_FOUND", Message: "文件未找到"}
	ErrPermissionDenied = &SystemError{Code: "PERMISSION_DENIED", Message: "权限不足"}
	ErrNetworkError     = &SystemError{Code: "NETWORK_ERROR", Message: "网络错误"}
	ErrTimeout          = &SystemError{Code: "TIMEOUT", Message: "操作超时"}
	ErrResourceBusy     = &SystemError{Code: "RESOURCE_BUSY", Message: "资源忙"}
	ErrAlreadyExists    = &SystemError{Code: "ALREADY_EXISTS", Message: "资源已存在"}
	ErrNotFound         = &SystemError{Code: "NOT_FOUND", Message: "资源未找到"}
)
