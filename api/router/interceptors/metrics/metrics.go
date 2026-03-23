package metrics

import (
	"api/router"
	"sync/atomic"
	"time"
)

// 指标拦截器
type MetricsInterceptor struct {
	callCount    int64 // 调用次数
	successCount int64 // 成功次数
	errorCount   int64 // 错误次数
	totalTime    int64 // 总耗时（纳秒）
	lastCallTime int64 // 最后调用时间（Unix纳秒）
}

// 指标拦截器
func New() *MetricsInterceptor {
	return &MetricsInterceptor{}
}

// 实现接口
func (mi *MetricsInterceptor) Intercept(ctx *router.Context, call *router.InterceptorCall) error {
	if call.Result == nil && call.Err == nil {
		return nil
	}
	atomic.AddInt64(&mi.callCount, 1)
	atomic.AddInt64(&mi.totalTime, int64(call.Duration))
	atomic.StoreInt64(&mi.lastCallTime, time.Now().UnixNano())
	if call.Err != nil {
		atomic.AddInt64(&mi.errorCount, 1)
	} else {
		atomic.AddInt64(&mi.successCount, 1)
	}
	return nil
}

// 函数统计
func (mi *MetricsInterceptor) Stats() map[string]any {
	callCount := atomic.LoadInt64(&mi.callCount)
	successCount := atomic.LoadInt64(&mi.successCount)
	errorCount := atomic.LoadInt64(&mi.errorCount)
	totalTime := atomic.LoadInt64(&mi.totalTime)
	lastCallTime := atomic.LoadInt64(&mi.lastCallTime)
	avgTime := time.Duration(0)
	if callCount > 0 {
		avgTime = time.Duration(totalTime / callCount)
	}
	errorRate := 0.0
	if callCount > 0 {
		errorRate = float64(errorCount) / float64(callCount)
	}
	successRate := 0.0
	if callCount > 0 {
		successRate = float64(successCount) / float64(callCount)
	}
	lastCall := time.Unix(0, lastCallTime)
	return map[string]any{
		"call_count":    callCount,
		"success_count": successCount,
		"error_count":   errorCount,
		"total_time_ns": totalTime,
		"avg_time":      avgTime.String(),
		"error_rate":    errorRate,
		"success_rate":  successRate,
		"last_call":     lastCall.Format("2006-01-02 15:04:05"),
		"qps":           mi.calculateQPS(),
	}
}

// 计算
func (mi *MetricsInterceptor) calculateQPS() float64 {
	callCount := atomic.LoadInt64(&mi.callCount)
	if callCount == 0 {
		return 0.0
	}
	lastCallTime := atomic.LoadInt64(&mi.lastCallTime)
	firstCallTime := mi.getFirstCallTime()
	if firstCallTime == 0 {
		return 0.0
	}
	duration := float64(lastCallTime-firstCallTime) / 1e9 // 转换为秒
	if duration <= 0 {
		return 0.0
	}
	return float64(callCount) / duration
}

// 第一次时间简化实现。
func (mi *MetricsInterceptor) getFirstCallTime() int64 {
	// 在实际实现中应该记录第一次时间。
	// 这里使用最后时间减去平均间隔作为近似。
	callCount := atomic.LoadInt64(&mi.callCount)
	if callCount == 0 {
		return 0
	}
	lastCallTime := atomic.LoadInt64(&mi.lastCallTime)
	totalTime := atomic.LoadInt64(&mi.totalTime)
	avgTime := totalTime / callCount
	// 估计第一次时间。
	return lastCallTime - (avgTime * callCount)
}

// 重置
func (mi *MetricsInterceptor) Reset() {
	atomic.StoreInt64(&mi.callCount, 0)
	atomic.StoreInt64(&mi.successCount, 0)
	atomic.StoreInt64(&mi.errorCount, 0)
	atomic.StoreInt64(&mi.totalTime, 0)
	atomic.StoreInt64(&mi.lastCallTime, 0)
}
