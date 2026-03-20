package system

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 网络系统实现（简化版）
// ============================================================================

// NetworkSystem 网络系统实现
type NetworkSystem struct {
	httpClient   *http.Client
	httpTimeout  time.Duration
	proxyURL     *url.URL
	tlsConfig    *tls.Config
	networkStats *NetworkStats
	statsMutex   sync.RWMutex
}

// NewNetworkSystem 创建新的网络系统
func NewNetworkSystem() *NetworkSystem {
	// 创建默认的HTTP客户端
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &NetworkSystem{
		httpClient:  client,
		httpTimeout: 30 * time.Second,
		networkStats: &NetworkStats{
			Interfaces:  0,
			BytesSent:   0,
			BytesRecv:   0,
			PacketsSent: 0,
			PacketsRecv: 0,
		},
	}
}

// ============================================================================
// HTTP客户端实现
// ============================================================================

// HTTPGet 执行HTTP GET请求
func (ns *NetworkSystem) HTTPGet(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error) {
	return ns.HTTPRequest(ctx, "GET", url, headers, nil)
}

// HTTPPost 执行HTTP POST请求
func (ns *NetworkSystem) HTTPPost(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	return ns.HTTPRequest(ctx, "POST", url, headers, body)
}

// HTTPPut 执行HTTP PUT请求
func (ns *NetworkSystem) HTTPPut(ctx context.Context, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	return ns.HTTPRequest(ctx, "PUT", url, headers, body)
}

// HTTPDelete 执行HTTP DELETE请求
func (ns *NetworkSystem) HTTPDelete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error) {
	return ns.HTTPRequest(ctx, "DELETE", url, headers, nil)
}

// HTTPRequest 执行HTTP请求
func (ns *NetworkSystem) HTTPRequest(ctx context.Context, method, urlStr string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	startTime := time.Now()

	// 创建请求
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, &SystemError{
			Code:    "INVALID_REQUEST",
			Message: fmt.Sprintf("创建请求失败: %v", err),
			Cause:   err,
		}
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 如果没有设置Content-Type，根据body类型自动设置
	if body != nil && req.Header.Get("Content-Type") == "" {
		// 尝试判断是否为JSON
		var jsonCheck interface{}
		if json.Unmarshal(body, &jsonCheck) == nil {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/octet-stream")
		}
	}

	// 设置User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "OpenCat-System/1.0")
	}

	// 执行请求
	resp, err := ns.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return nil, &SystemError{
			Code:    "NETWORK_ERROR",
			Message: fmt.Sprintf("HTTP请求失败: %v", err),
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &SystemError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("读取响应体失败: %v", err),
			Cause:   err,
		}
	}

	// 更新统计
	ns.statsMutex.Lock()
	ns.networkStats.BytesSent += len(body)
	ns.networkStats.BytesRecv += len(respBody)
	ns.statsMutex.Unlock()

	// 构建响应
	response := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string][]string),
		Body:       respBody,
		Duration:   duration,
	}

	// 复制响应头
	for key, values := range resp.Header {
		response.Headers[key] = values
	}

	return response, nil
}

// SetTimeout 设置HTTP超时时间
func (ns *NetworkSystem) SetTimeout(timeout time.Duration) {
	ns.httpTimeout = timeout
	ns.httpClient.Timeout = timeout
}

// SetProxy 设置代理
func (ns *NetworkSystem) SetProxy(proxyURL string) error {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return &SystemError{
			Code:    "INVALID_PROXY",
			Message: fmt.Sprintf("代理URL无效: %v", err),
			Cause:   err,
		}
	}

	ns.proxyURL = parsedURL
	if transport, ok := ns.httpClient.Transport.(*http.Transport); ok {
		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return nil
}

// SetTLSConfig 设置TLS配置
func (ns *NetworkSystem) SetTLSConfig(config *TLSConfig) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	// 加载证书文件
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return &SystemError{
				Code:    "TLS_ERROR",
				Message: fmt.Sprintf("加载证书失败: %v", err),
				Cause:   err,
			}
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载CA证书
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return &SystemError{
				Code:    "TLS_ERROR",
				Message: fmt.Sprintf("加载CA证书失败: %v", err),
				Cause:   err,
			}
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return &SystemError{
				Code:    "TLS_ERROR",
				Message: "添加CA证书失败",
			}
		}
		tlsConfig.RootCAs = caCertPool
	}

	ns.tlsConfig = tlsConfig
	if transport, ok := ns.httpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = tlsConfig
	}

	return nil
}

// ============================================================================
// WebSocket客户端实现（简化版）
// ============================================================================

// WebSocket连接管理器
type websocketManager struct {
	connections map[string]*websocketConnection
	mutex       sync.RWMutex
}

// WebSocket连接
type websocketConnection struct {
	id        string
	url       string
	connected bool
	createdAt time.Time
	lastPing  time.Time
	messages  chan []byte
	closeChan chan struct{}
}

// 全局WebSocket管理器
var wsManager = &websocketManager{
	connections: make(map[string]*websocketConnection),
}

// WebSocketConnect 建立WebSocket连接
func (ns *NetworkSystem) WebSocketConnect(ctx context.Context, url string, headers map[string]string) (*WebSocketConnection, error) {
	// 生成连接ID
	connID := fmt.Sprintf("ws_%d_%s", time.Now().UnixNano(), url)

	// 创建连接对象
	conn := &websocketConnection{
		id:        connID,
		url:       url,
		connected: true,
		createdAt: time.Now(),
		lastPing:  time.Now(),
		messages:  make(chan []byte, 100),
		closeChan: make(chan struct{}),
	}

	// 保存连接
	wsManager.mutex.Lock()
	wsManager.connections[connID] = conn
	wsManager.mutex.Unlock()

	// 启动心跳协程
	go ns.websocketHeartbeat(conn)

	// 返回连接信息
	return &WebSocketConnection{
		ID:        connID,
		URL:       url,
		Connected: true,
		CreatedAt: conn.createdAt,
		LastPing:  conn.lastPing,
	}, nil
}

// WebSocketClose 关闭WebSocket连接
func (ns *NetworkSystem) WebSocketClose(connID string) error {
	wsManager.mutex.Lock()
	defer wsManager.mutex.Unlock()

	conn, exists := wsManager.connections[connID]
	if !exists {
		return &SystemError{
			Code:    "WS_CONNECTION_NOT_FOUND",
			Message: fmt.Sprintf("WebSocket连接不存在: %s", connID),
		}
	}

	// 关闭连接
	close(conn.closeChan)
	conn.connected = false

	// 移除连接
	delete(wsManager.connections, connID)

	return nil
}

// WebSocketSend 发送WebSocket消息
func (ns *NetworkSystem) WebSocketSend(connID string, message []byte) error {
	wsManager.mutex.RLock()
	conn, exists := wsManager.connections[connID]
	wsManager.mutex.RUnlock()

	if !exists {
		return &SystemError{
			Code:    "WS_CONNECTION_NOT_FOUND",
			Message: fmt.Sprintf("WebSocket连接不存在: %s", connID),
		}
	}

	if !conn.connected {
		return &SystemError{
			Code:    "WS_CONNECTION_CLOSED",
			Message: "WebSocket连接已关闭",
		}
	}

	// 简化实现：将消息放入队列
	select {
	case conn.messages <- message:
		return nil
	default:
		return &SystemError{
			Code:    "WS_SEND_FAILED",
			Message: "发送消息失败：队列已满",
		}
	}
}

// WebSocketReceive 接收WebSocket消息
func (ns *NetworkSystem) WebSocketReceive(connID string, timeout time.Duration) ([]byte, error) {
	wsManager.mutex.RLock()
	conn, exists := wsManager.connections[connID]
	wsManager.mutex.RUnlock()

	if !exists {
		return nil, &SystemError{
			Code:    "WS_CONNECTION_NOT_FOUND",
			Message: fmt.Sprintf("WebSocket连接不存在: %s", connID),
		}
	}

	if !conn.connected {
		return nil, &SystemError{
			Code:    "WS_CONNECTION_CLOSED",
			Message: "WebSocket连接已关闭",
		}
	}

	// 设置超时
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case message := <-conn.messages:
		return message, nil
	case <-timer.C:
		return nil, &SystemError{
			Code:    "WS_RECEIVE_TIMEOUT",
			Message: "接收消息超时",
		}
	case <-conn.closeChan:
		return nil, &SystemError{
			Code:    "WS_CONNECTION_CLOSED",
			Message: "WebSocket连接已关闭",
		}
	}
}

// WebSocketStatus 获取WebSocket连接状态
func (ns *NetworkSystem) WebSocketStatus(connID string) (*WebSocketStatus, error) {
	wsManager.mutex.RLock()
	conn, exists := wsManager.connections[connID]
	wsManager.mutex.RUnlock()

	if !exists {
		return nil, &SystemError{
			Code:    "WS_CONNECTION_NOT_FOUND",
			Message: fmt.Sprintf("WebSocket连接不存在: %s", connID),
		}
	}

	// 计算消息数量
	messageCount := len(conn.messages)

	return &WebSocketStatus{
		Connected:    conn.connected,
		MessagesSent: 0, // 简化实现
		MessagesRecv: messageCount,
		LastActivity: time.Now(),
		ErrorCount:   0,
	}, nil
}

// ListWebSocketConnections 列出所有WebSocket连接
func (ns *NetworkSystem) ListWebSocketConnections() []string {
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	connections := make([]string, 0, len(wsManager.connections))
	for connID := range wsManager.connections {
		connections = append(connections, connID)
	}

	return connections
}

// websocketHeartbeat WebSocket心跳
func (ns *NetworkSystem) websocketHeartbeat(conn *websocketConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 更新最后心跳时间
			conn.lastPing = time.Now()

			// 发送心跳消息
			heartbeatMsg := []byte("ping")
			select {
			case conn.messages <- heartbeatMsg:
				// 心跳发送成功
			default:
				// 队列已满，忽略
			}

		case <-conn.closeChan:
			return
		}
	}
}

// ============================================================================
// 网络监控实现
// ============================================================================

// Ping 执行ping测试
func (ns *NetworkSystem) Ping(ctx context.Context, host string, count int) (*PingResult, error) {
	// 简化实现：使用net.Dial测试连通性
	var successes int
	var totalRTT time.Duration
	var minRTT, maxRTT time.Duration

	for i := 0; i < count; i++ {
		connStart := time.Now()
		conn, err := net.DialTimeout("tcp", host+":80", 5*time.Second)
		rtt := time.Since(connStart)

		if err == nil {
			successes++
			totalRTT += rtt

			if i == 0 || rtt < minRTT {
				minRTT = rtt
			}
			if rtt > maxRTT {
				maxRTT = rtt
			}

			conn.Close()
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 等待一段时间再进行下一次ping
		if i < count-1 {
			time.Sleep(1 * time.Second)
		}
	}

	packetLoss := float64(count-successes) / float64(count) * 100
	avgRTT := time.Duration(0)
	if successes > 0 {
		avgRTT = totalRTT / time.Duration(successes)
	}

	return &PingResult{
		Host:        host,
		PacketsSent: count,
		PacketsRecv: successes,
		PacketLoss:  packetLoss,
		MinRTT:      minRTT,
		MaxRTT:      maxRTT,
		AvgRTT:      avgRTT,
		StdDevRTT:   0, // 简化实现，不计算标准差
	}, nil
}

// Traceroute 执行traceroute测试
func (ns *NetworkSystem) Traceroute(ctx context.Context, host string) (*TracerouteResult, error) {
	// 简化实现：返回空结果
	return &TracerouteResult{
		Host:    host,
		Hops:    []TracerouteHop{},
		Success: false,
	}, ErrNotImplemented
}

// DNSLookup 执行DNS查询
func (ns *NetworkSystem) DNSLookup(ctx context.Context, host string) (*DNSResult, error) {
	// 执行DNS查询
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, &SystemError{
			Code:    "DNS_ERROR",
			Message: fmt.Sprintf("DNS查询失败: %v", err),
			Cause:   err,
		}
	}

	return &DNSResult{
		Host:    host,
		Records: addrs,
	}, nil
}

// GetNetworkInterfaces 获取网络接口信息
func (ns *NetworkSystem) GetNetworkInterfaces() ([]NetworkInterface, error) {
	// 获取网络接口列表
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, &SystemError{
			Code:    "NETWORK_ERROR",
			Message: fmt.Sprintf("获取网络接口失败: %v", err),
			Cause:   err,
		}
	}

	interfaces := make([]NetworkInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		// 获取接口地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		addrStrings := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		// 获取接口标志
		flags := make([]string, 0)
		if iface.Flags&net.FlagUp != 0 {
			flags = append(flags, "up")
		}
		if iface.Flags&net.FlagLoopback != 0 {
			flags = append(flags, "loopback")
		}
		if iface.Flags&net.FlagBroadcast != 0 {
			flags = append(flags, "broadcast")
		}
		if iface.Flags&net.FlagPointToPoint != 0 {
			flags = append(flags, "pointtopoint")
		}
		if iface.Flags&net.FlagMulticast != 0 {
			flags = append(flags, "multicast")
		}

		interfaces = append(interfaces, NetworkInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr.String(),
			Flags:        flags,
			Addrs:        addrStrings,
		})
	}

	// 更新统计
	ns.statsMutex.Lock()
	ns.networkStats.Interfaces = len(interfaces)
	ns.statsMutex.Unlock()

	return interfaces, nil
}

// GetNetworkStats 获取网络统计信息
func (ns *NetworkSystem) GetNetworkStats() (*NetworkStats, error) {
	ns.statsMutex.RLock()
	defer ns.statsMutex.RUnlock()

	// 返回统计信息的副本
	stats := *ns.networkStats
	return &stats, nil
}
