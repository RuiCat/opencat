package auth

import (
	"api/router"
	"fmt"
)

// AuthInterceptor 认证拦截器
type AuthInterceptor struct {
	validator func(*router.Context) error // 验证函数
	required  []string                    // 需要的权限或角色
}

// New 创建认证拦截器
// validator: 自定义验证函数
// required: 需要的权限或角色
// 返回: 认证拦截器实例
func New(validator func(*router.Context) error, required ...string) *AuthInterceptor {
	return &AuthInterceptor{
		validator: validator,
		required:  required,
	}
}

// Intercept 实现拦截器接口
// ctx: 执行上下文
// call: 拦截器调用信息
// 返回: 错误信息
func (ai *AuthInterceptor) Intercept(ctx *router.Context, call *router.InterceptorCall) error {
	if call.Result != nil || call.Err != nil {
		return nil
	}
	if ai.validator != nil {
		return ai.validator(ctx)
	}
	return ai.defaultValidate(ctx)
}

// defaultValidate 默认验证逻辑
// ctx: 执行上下文
// 返回: 验证错误
func (ai *AuthInterceptor) defaultValidate(ctx *router.Context) error {
	// 是否有用户
	user := ctx.GetValue("user")
	if user == nil {
		return fmt.Errorf("未认证: 上下文缺少用户信息")
	}
	if len(ai.required) > 0 {
		roles := ctx.GetValue("roles")
		if roles == nil {
			return fmt.Errorf("未授权: 用户缺少角色信息")
		}
		roleList, ok := roles.([]string)
		if !ok {
			return fmt.Errorf("未授权: 角色信息格式错误")
		}
		if !hasRequiredRole(roleList, ai.required) {
			return fmt.Errorf("未授权: 用户缺少必要权限")
		}
	}
	return nil
}

// hasRequiredRole 检查是否有需要的角色
// userRoles: 用户角色列表
// requiredRoles: 需要的角色列表
// 返回: 是否满足要求
func hasRequiredRole(userRoles, requiredRoles []string) bool {
	roleMap := make(map[string]bool)
	for _, role := range userRoles {
		roleMap[role] = true
	}
	for _, required := range requiredRoles {
		if !roleMap[required] {
			return false
		}
	}
	return true
}

// SimpleAuth 创建简单认证拦截器
// 返回: 认证拦截器实例
func SimpleAuth() *AuthInterceptor {
	return New(nil)
}

// RoleAuth 创建角色认证拦截器
// roles: 需要的角色
// 返回: 认证拦截器实例
func RoleAuth(roles ...string) *AuthInterceptor {
	return New(nil, roles...)
}
