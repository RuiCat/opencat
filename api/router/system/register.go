package system

import (
	"api/router"
	"context"
	"time"
)

// ============================================================================
// 系统接口注册器
// ============================================================================

// RegisterSystemFunctions 注册系统函数到路由
func RegisterSystemFunctions(r *router.Router) error {
	sysMgr := NewSystemManager()

	// 注册路由管理函数
	if err := registerRouterFunctions(r, sysMgr); err != nil {
		return err
	}

	// 注册网络函数
	if err := registerNetworkFunctions(r, sysMgr); err != nil {
		return err
	}

	// 注册文件系统函数
	if err := registerFilesystemFunctions(r, sysMgr); err != nil {
		return err
	}

	// 注册系统信息函数
	if err := registerSystemInfoFunctions(r, sysMgr); err != nil {
		return err
	}

	// 注册工具函数
	if err := registerUtilityFunctions(r, sysMgr); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// 网络函数注册
// ============================================================================

func registerNetworkFunctions(r *router.Router, sysMgr *SystemManager) error {
	// HTTP GET 函数
	r.Register(&router.Function{
		Name:        "system.http.get",
		Description: "执行HTTP GET请求",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求URL",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "请求头",
				},
			},
			"required": []string{"url"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			url, ok := block.Payload["url"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 url 必须是字符串",
					},
				}
			}

			headers := make(map[string]string)
			if h, ok := block.Payload["headers"].(map[string]interface{}); ok {
				for k, v := range h {
					if str, ok := v.(string); ok {
						headers[k] = str
					}
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			resp, err := sysMgr.HTTPGet(context.Background(), url, headers)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "HTTP_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"status_code": resp.StatusCode,
					"headers":     resp.Headers,
					"body":        string(resp.Body),
					"duration":    resp.Duration.String(),
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// HTTP POST 函数
	r.Register(&router.Function{
		Name:        "system.http.post",
		Description: "执行HTTP POST请求",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求URL",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "请求头",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "请求体",
				},
			},
			"required": []string{"url", "body"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			url, ok := block.Payload["url"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 url 必须是字符串",
					},
				}
			}

			body, ok := block.Payload["body"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 body 必须是字符串",
					},
				}
			}

			headers := make(map[string]string)
			if h, ok := block.Payload["headers"].(map[string]interface{}); ok {
				for k, v := range h {
					if str, ok := v.(string); ok {
						headers[k] = str
					}
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			resp, err := sysMgr.HTTPPost(context.Background(), url, headers, []byte(body))
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "HTTP_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"status_code": resp.StatusCode,
					"headers":     resp.Headers,
					"body":        string(resp.Body),
					"duration":    resp.Duration.String(),
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// Ping 函数
	r.Register(&router.Function{
		Name:        "system.network.ping",
		Description: "执行网络ping测试",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "目标主机",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "ping次数",
					"default":     4,
				},
			},
			"required": []string{"host"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			host, ok := block.Payload["host"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 host 必须是字符串",
					},
				}
			}

			count := 4
			if c, ok := block.Payload["count"].(float64); ok {
				count = int(c)
			}

			// 使用 context.Background() 替代 ctx.Context
			result, err := sysMgr.Ping(context.Background(), host, count)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "PING_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"host":         result.Host,
					"packets_sent": result.PacketsSent,
					"packets_recv": result.PacketsRecv,
					"packet_loss":  result.PacketLoss,
					"min_rtt":      result.MinRTT.String(),
					"max_rtt":      result.MaxRTT.String(),
					"avg_rtt":      result.AvgRTT.String(),
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// DNS查询函数
	r.Register(&router.Function{
		Name:        "system.network.dns",
		Description: "执行DNS查询",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "查询的主机名",
				},
			},
			"required": []string{"host"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			host, ok := block.Payload["host"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 host 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			result, err := sysMgr.DNSLookup(context.Background(), host)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "DNS_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"host":    result.Host,
					"records": result.Records,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	return nil
}

// ============================================================================
// 文件系统函数注册
// ============================================================================

func registerFilesystemFunctions(r *router.Router, sysMgr *SystemManager) error {
	// 读取文件函数
	r.Register(&router.Function{
		Name:        "system.fs.read",
		Description: "读取文件内容",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
			},
			"required": []string{"path"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			path, ok := block.Payload["path"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 path 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			data, err := sysMgr.FileRead(context.Background(), path)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "READ_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"content": string(data),
					"size":    len(data),
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 写入文件函数
	r.Register(&router.Function{
		Name:        "system.fs.write",
		Description: "写入文件内容",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文件内容",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "是否追加",
					"default":     false,
				},
			},
			"required": []string{"path", "content"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			path, ok := block.Payload["path"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 path 必须是字符串",
					},
				}
			}

			content, ok := block.Payload["content"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 content 必须是字符串",
					},
				}
			}

			append := false
			if a, ok := block.Payload["append"].(bool); ok {
				append = a
			}

			// 使用 context.Background() 替代 ctx.Context
			err := sysMgr.FileWrite(context.Background(), path, []byte(content), append)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "WRITE_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"message": "文件写入成功",
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 列出目录函数
	r.Register(&router.Function{
		Name:        "system.fs.list",
		Description: "列出目录内容",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			path, ok := block.Payload["path"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 path 必须是字符串",
					},
				}
			}

			recursive := false
			if r, ok := block.Payload["recursive"].(bool); ok {
				recursive = r
			}

			// 使用 context.Background() 替代 ctx.Context
			files, err := sysMgr.DirList(context.Background(), path, recursive)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "LIST_ERROR",
						Message: err.Error(),
					},
				}
			}

			result := make([]map[string]interface{}, len(files))
			for i, file := range files {
				result[i] = map[string]interface{}{
					"name":     file.Name,
					"path":     file.Path,
					"size":     file.Size,
					"mode":     file.Mode,
					"mod_time": file.ModTime.Format(time.RFC3339),
					"is_dir":   file.IsDir,
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"files": result,
					"count": len(result),
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 检查文件是否存在函数
	r.Register(&router.Function{
		Name:        "system.fs.exists",
		Description: "检查文件或目录是否存在",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "路径",
				},
			},
			"required": []string{"path"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			path, ok := block.Payload["path"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 path 必须是字符串",
					},
				}
			}

			// 使用 context.Background() 替代 ctx.Context
			exists, err := sysMgr.FileExists(context.Background(), path)
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "STAT_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"exists": exists,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	return nil
}

// ============================================================================
// 系统信息函数注册
// ============================================================================

func registerSystemInfoFunctions(r *router.Router, sysMgr *SystemManager) error {
	// 获取系统信息函数
	r.Register(&router.Function{
		Name:        "system.info.get",
		Description: "获取系统信息",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			info, err := sysMgr.GetSystemInfo()
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "SYSTEM_INFO_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"hostname":      info.Hostname,
					"os":            info.OS,
					"arch":          info.Arch,
					"kernel":        info.Kernel,
					"uptime":        info.Uptime,
					"load_avg":      info.LoadAvg,
					"num_cpu":       info.NumCPU,
					"num_goroutine": info.NumGoroutine,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 获取CPU信息函数
	r.Register(&router.Function{
		Name:        "system.info.cpu",
		Description: "获取CPU信息",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			cpuInfo, err := sysMgr.GetCPUInfo()
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "CPU_INFO_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"model":       cpuInfo.Model,
					"cores":       cpuInfo.Cores,
					"usage":       cpuInfo.Usage,
					"user":        cpuInfo.User,
					"system":      cpuInfo.System,
					"idle":        cpuInfo.Idle,
					"temperature": cpuInfo.Temperature,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 获取内存信息函数
	r.Register(&router.Function{
		Name:        "system.info.memory",
		Description: "获取内存信息",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			memInfo, err := sysMgr.GetMemoryInfo()
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "MEMORY_INFO_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"total":     memInfo.Total,
					"used":      memInfo.Used,
					"free":      memInfo.Free,
					"available": memInfo.Available,
					"usage":     memInfo.Usage,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 获取磁盘信息函数
	r.Register(&router.Function{
		Name:        "system.info.disk",
		Description: "获取磁盘信息",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			diskInfo, err := sysMgr.GetDiskInfo()
			if err != nil {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "DISK_INFO_ERROR",
						Message: err.Error(),
					},
				}
			}

			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"total": diskInfo.Total,
					"used":  diskInfo.Used,
					"free":  diskInfo.Free,
					"usage": diskInfo.Usage,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	return nil
}

// ============================================================================
// 工具函数注册
// ============================================================================

func registerUtilityFunctions(r *router.Router, sysMgr *SystemManager) error {
	// Base64编码函数
	r.Register(&router.Function{
		Name:        "system.util.base64_encode",
		Description: "Base64编码",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":        "string",
					"description": "要编码的数据",
				},
			},
			"required": []string{"data"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			data, ok := block.Payload["data"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 data 必须是字符串",
					},
				}
			}

			encoded := sysMgr.Base64Encode([]byte(data))
			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"encoded": encoded,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// MD5哈希函数
	r.Register(&router.Function{
		Name:        "system.util.md5",
		Description: "计算MD5哈希",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":        "string",
					"description": "要计算哈希的数据",
				},
			},
			"required": []string{"data"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			data, ok := block.Payload["data"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 data 必须是字符串",
					},
				}
			}

			hash := sysMgr.MD5([]byte(data))
			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"hash": hash,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// SHA256哈希函数
	r.Register(&router.Function{
		Name:        "system.util.sha256",
		Description: "计算SHA256哈希",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":        "string",
					"description": "要计算哈希的数据",
				},
			},
			"required": []string{"data"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			data, ok := block.Payload["data"].(string)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 data 必须是字符串",
					},
				}
			}

			hash := sysMgr.SHA256([]byte(data))
			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"hash": hash,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 生成随机字符串函数
	r.Register(&router.Function{
		Name:        "system.util.random_string",
		Description: "生成随机字符串",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"length": map[string]interface{}{
					"type":        "integer",
					"description": "字符串长度",
					"default":     16,
				},
			},
			"required": []string{"length"},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			length, ok := block.Payload["length"].(float64)
			if !ok {
				return &router.Result{
					Success: false,
					Error: &router.ErrorInfo{
						Code:    "INVALID_PARAMETER",
						Message: "参数 length 必须是整数",
					},
				}
			}

			randomStr := sysMgr.RandomString(int(length))
			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"random_string": randomStr,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	// 生成UUID函数
	r.Register(&router.Function{
		Name:        "system.util.uuid",
		Description: "生成UUID",
		Namespace:   "system",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx *router.Context, block *router.DataBlock) *router.Result {
			uuid := sysMgr.UUID()
			return &router.Result{
				Success: true,
				Data: map[string]interface{}{
					"uuid": uuid,
				},
			}
		},
		Builtin:   true,
		Enabled:   true,
		CreatedAt: time.Now(),
	})

	return nil
}
