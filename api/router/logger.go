package router

import (
	"sync"
	"time"
)

// Logger 日志记录器
type Logger struct {
	mu      sync.RWMutex
	entries []LogEntry
	maxSize int
}

// NewLogger 创建新的日志记录器
func NewLogger() *Logger {
	return &Logger{
		entries: make([]LogEntry, 0),
		maxSize: 1000, // 最多保存1000条日志
	}
}

// Log 记录日志
func (l *Logger) Log(level, message string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Fields:    fields,
	}

	l.entries = append(l.entries, entry)

	// 限制日志大小
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

// GetEntries 获取日志条目
func (l *Logger) GetEntries(level string, limit int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, 0)
	count := 0

	// 从最新开始遍历
	for i := len(l.entries) - 1; i >= 0; i-- {
		entry := l.entries[i]
		if level == "" || entry.Level == level {
			result = append(result, entry)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	// 反转结果，使最新的在前
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// Clear 清空日志
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0)
}

// GetStats 获取日志统计
func (l *Logger) GetStats() map[string]int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := make(map[string]int)
	for _, entry := range l.entries {
		stats[entry.Level]++
	}
	stats["total"] = len(l.entries)

	return stats
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	mu      sync.RWMutex
	logs    []AuditLog
	maxSize int
}

// AuditLog 审计日志
type AuditLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	AgentID   string                 `json:"agent_id"`
	Target    string                 `json:"target"`
	Success   bool                   `json:"success"`
	Duration  time.Duration          `json:"duration_ms"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logs:    make([]AuditLog, 0),
		maxSize: 5000, // 最多保存5000条审计日志
	}
}

// Log 记录审计日志
func (al *AuditLogger) Log(action, agentID, target string, success bool, duration time.Duration, err error, details map[string]interface{}) {
	al.mu.Lock()
	defer al.mu.Unlock()

	log := AuditLog{
		ID:        generateID(),
		Timestamp: time.Now(),
		Action:    action,
		AgentID:   agentID,
		Target:    target,
		Success:   success,
		Duration:  duration,
		Details:   details,
	}

	if err != nil {
		log.Error = err.Error()
	}

	al.logs = append(al.logs, log)

	// 限制日志大小
	if len(al.logs) > al.maxSize {
		al.logs = al.logs[len(al.logs)-al.maxSize:]
	}
}

// Query 查询审计日志
func (al *AuditLogger) Query(filter AuditLogFilter) []AuditLog {
	al.mu.RLock()
	defer al.mu.RUnlock()

	result := make([]AuditLog, 0)
	for _, log := range al.logs {
		if filter.Match(log) {
			result = append(result, log)
		}
	}

	return result
}

// AuditLogFilter 审计日志过滤器
type AuditLogFilter struct {
	Action   string
	AgentID  string
	Target   string
	Success  *bool
	FromTime *time.Time
	ToTime   *time.Time
	Limit    int
}

// Match 检查日志是否匹配过滤器
func (f *AuditLogFilter) Match(log AuditLog) bool {
	if f.Action != "" && log.Action != f.Action {
		return false
	}
	if f.AgentID != "" && log.AgentID != f.AgentID {
		return false
	}
	if f.Target != "" && log.Target != f.Target {
		return false
	}
	if f.Success != nil && log.Success != *f.Success {
		return false
	}
	if f.FromTime != nil && log.Timestamp.Before(*f.FromTime) {
		return false
	}
	if f.ToTime != nil && log.Timestamp.After(*f.ToTime) {
		return false
	}
	return true
}

// GetStats 获取审计日志统计
func (al *AuditLogger) GetStats() map[string]interface{} {
	al.mu.RLock()
	defer al.mu.RUnlock()

	stats := map[string]interface{}{
		"total":     len(al.logs),
		"success":   0,
		"failure":   0,
		"by_action": make(map[string]int),
		"by_agent":  make(map[string]int),
		"by_target": make(map[string]int),
	}

	for _, log := range al.logs {
		if log.Success {
			stats["success"] = stats["success"].(int) + 1
		} else {
			stats["failure"] = stats["failure"].(int) + 1
		}

		stats["by_action"].(map[string]int)[log.Action]++
		stats["by_agent"].(map[string]int)[log.AgentID]++
		stats["by_target"].(map[string]int)[log.Target]++
	}

	return stats
}
