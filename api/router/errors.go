package router

// 错误代码定义
const (
	// 通用错误
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeInvalidParameter  = "INVALID_PARAMETER"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeAlreadyExists     = "ALREADY_EXISTS"
	ErrCodePermissionDenied  = "PERMISSION_DENIED"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrCodeResourceExhausted = "RESOURCE_EXHAUSTED"

	// 路由相关错误
	ErrCodeFunctionNotFound = "FUNCTION_NOT_FOUND"
	ErrCodeFunctionDisabled = "FUNCTION_DISABLED"
	ErrCodeMaxCallDepth     = "MAX_CALL_DEPTH_EXCEEDED"
	ErrCodeConcurrentLimit  = "CONCURRENT_LIMIT_EXCEEDED"
	ErrCodeMaxFunctions     = "MAX_FUNCTIONS_EXCEEDED"
	ErrCodeBuiltinFunction  = "BUILTIN_FUNCTION"
	ErrCodeInterceptorError = "INTERCEPTOR_ERROR"

	// 执行相关错误
	ErrCodePanicRecovered   = "PANIC_RECOVERED"
	ErrCodeExecutionFailed  = "EXECUTION_FAILED"
	ErrCodeValidationFailed = "VALIDATION_FAILED"

	// 网络相关错误
	ErrCodeNetworkError     = "NETWORK_ERROR"
	ErrCodeConnectionFailed = "CONNECTION_FAILED"

	// 文件系统相关错误
	ErrCodeFileNotFound    = "FILE_NOT_FOUND"
	ErrCodePermissionError = "PERMISSION_ERROR"
	ErrCodeDiskFull        = "DISK_FULL"
	ErrCodeIOError         = "IO_ERROR"
)

// RouterError 路由错误
type RouterError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

// Error 实现 error 接口
func (e *RouterError) Error() string {
	return e.Message
}

// NewRouterError 创建新的路由错误
func NewRouterError(code, message string) *RouterError {
	return &RouterError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail 添加错误详情
func (e *RouterError) WithDetail(key string, value interface{}) *RouterError {
	e.Details[key] = value
	return e
}

// 预定义的错误
var (
	// 通用错误
	ErrInternalError     = NewRouterError(ErrCodeInternalError, "内部错误")
	ErrInvalidParameter  = NewRouterError(ErrCodeInvalidParameter, "参数无效")
	ErrNotFound          = NewRouterError(ErrCodeNotFound, "资源未找到")
	ErrAlreadyExists     = NewRouterError(ErrCodeAlreadyExists, "资源已存在")
	ErrPermissionDenied  = NewRouterError(ErrCodePermissionDenied, "权限不足")
	ErrTimeout           = NewRouterError(ErrCodeTimeout, "操作超时")
	ErrRateLimitExceeded = NewRouterError(ErrCodeRateLimitExceeded, "速率限制超出")
	ErrResourceExhausted = NewRouterError(ErrCodeResourceExhausted, "资源耗尽")

	// 路由错误
	ErrFunctionNotFound = NewRouterError(ErrCodeFunctionNotFound, "函数未找到")
	ErrFunctionDisabled = NewRouterError(ErrCodeFunctionDisabled, "函数已禁用")
	ErrMaxCallDepth     = NewRouterError(ErrCodeMaxCallDepth, "调用深度超过限制")
	ErrConcurrentLimit  = NewRouterError(ErrCodeConcurrentLimit, "并发调用数达到限制")
	ErrMaxFunctions     = NewRouterError(ErrCodeMaxFunctions, "达到最大函数数量限制")
	ErrBuiltinFunction  = NewRouterError(ErrCodeBuiltinFunction, "不能操作内置函数")
	ErrInterceptorError = NewRouterError(ErrCodeInterceptorError, "拦截器错误")

	// 执行错误
	ErrPanicRecovered   = NewRouterError(ErrCodePanicRecovered, "函数执行panic已恢复")
	ErrExecutionFailed  = NewRouterError(ErrCodeExecutionFailed, "函数执行失败")
	ErrValidationFailed = NewRouterError(ErrCodeValidationFailed, "参数验证失败")

	// 网络错误
	ErrNetworkError     = NewRouterError(ErrCodeNetworkError, "网络错误")
	ErrConnectionFailed = NewRouterError(ErrCodeConnectionFailed, "连接失败")

	// 文件系统错误
	ErrFileNotFound    = NewRouterError(ErrCodeFileNotFound, "文件未找到")
	ErrPermissionError = NewRouterError(ErrCodePermissionError, "文件权限错误")
	ErrDiskFull        = NewRouterError(ErrCodeDiskFull, "磁盘空间不足")
	ErrIOError         = NewRouterError(ErrCodeIOError, "IO错误")
)

// IsRouterError 检查是否为路由错误
func IsRouterError(err error) bool {
	_, ok := err.(*RouterError)
	return ok
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) string {
	if routerErr, ok := err.(*RouterError); ok {
		return routerErr.Code
	}
	return ErrCodeInternalError
}

// WrapError 包装错误
func WrapError(code string, err error) *RouterError {
	if routerErr, ok := err.(*RouterError); ok {
		return routerErr
	}
	return NewRouterError(code, err.Error())
}

// ErrorToResult 将错误转换为结果
func ErrorToResult(err error) *Result {
	if routerErr, ok := err.(*RouterError); ok {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    routerErr.Code,
				Message: routerErr.Message,
			},
		}
	}

	return &Result{
		Success: false,
		Error: &ErrorInfo{
			Code:    ErrCodeInternalError,
			Message: err.Error(),
		},
	}
}

// SuccessResult 创建成功结果
func SuccessResult(data interface{}) *Result {
	return &Result{
		Success: true,
		Data:    data,
	}
}

// ErrorResult 创建错误结果
func ErrorResult(code, message string) *Result {
	return &Result{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}
