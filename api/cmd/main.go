package main

import (
	"api/router"
	"api/router/interceptors/auth"
	"api/router/interceptors/logging"
	"api/router/interceptors/metrics"
	"fmt"
	"time"
)

// 表示数学结构。
type Math struct {
	S int
}

func main() {
	fmt.Println("=== 测试拦截器框架 ===")

	// 增强
	r := router.NewRouter(nil)
	fmt.Println("1. 增强路由器创建完成")
	ctx := router.NewContext("test-session", "test-agent", r)

	logger := logging.SimpleLogger()
	authenticator := auth.RoleAuth("admin", "user")
	metricCollector := metrics.New()

	// 全局
	ctx.AddGlobalInterceptor(logger, 1000)
	ctx.AddGlobalInterceptor(authenticator, 500)
	fmt.Printf("2. 全局拦截器添加完成，数量: %d\n", ctx.GlobalInterceptorCount())

	mathFunc := func(math *Math, a, b int) int {
		fmt.Printf("    [函数执行] 计算: %d + %d + %d\n", a, b, math.S)
		time.Sleep(50 * time.Millisecond) // 模拟耗时操作
		return a + b + math.S
	}

	err := ctx.RegisterWithInterceptors(
		"add",
		"加法函数",
		"math",
		[]string{"a", "b"},
		[]string{"result"},
		mathFunc,
		metricCollector, // 函数特定拦截器
	)

	if err != nil {
		fmt.Printf("注册函数失败: %v\n", err)
		return
	}
	fmt.Println("3. 函数注册完成（带拦截器）")

	ctx.SetValue("math", &Math{S: 100})

	// 用户和角色用于。
	ctx.SetValue("user", "test-user")
	ctx.SetValue("roles", []string{"admin", "user"})

	fmt.Println("4. 上下文创建完成")

	// 测试正常
	fmt.Println("\n5. 测试正常调用:")
	for i := 1; i <= 3; i++ {
		result, err := router.CallEnhanced(ctx, "add", i, i*2)
		if err != nil {
			fmt.Printf("   调用 %d 失败: %v\n", i, err)
		} else {
			fmt.Printf("   调用 %d 结果: %v\n", i, result)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 测试
	fmt.Println("\n6. 测试认证失败:")
	ctx2 := router.NewContext("test-session-2", "test-agent-2", r)
	ctx2.SetValue("math", &Math{S: 100})
	// 不用户应该

	result, err := router.CallEnhanced(ctx2, "add", 1, 2)
	if err != nil {
		fmt.Printf("   认证失败（预期）: %v\n", err)
	} else {
		fmt.Printf("   意外成功: %v\n", result)
	}

	// 查看
	fmt.Println("\n7. 查看指标统计:")
	stats := metricCollector.Stats()
	for key, value := range stats {
		fmt.Printf("   %s: %v\n", key, value)
	}

	// 测试动态
	fmt.Println("\n8. 测试动态添加拦截器:")

	// 新的
	debugLogger := logging.DebugLogger()

	// 动态到
	err = ctx.AddFunctionInterceptor("add", debugLogger, 100)
	if err != nil {
		fmt.Printf("   动态添加拦截器失败: %v\n", err)
	} else {
		fmt.Printf("   动态添加拦截器成功，函数拦截器数量: %d\n", ctx.FunctionInterceptorCount("add"))

		// 测试带新的
		result, err = router.CallEnhanced(ctx, "add", 10, 20)
		if err != nil {
			fmt.Printf("   调用失败: %v\n", err)
		} else {
			fmt.Printf("   调用结果: %v\n", result)
		}
	}

	// 测试链式
	fmt.Println("\n9. 测试链式调用:")
	router.CallFuncEnhanced(ctx, "add", []interface{}{100, 200}, func(result interface{}, err error) {
		if err != nil {
			fmt.Printf("   链式调用失败: %v\n", err)
		} else {
			fmt.Printf("   链式调用结果: %v\n", result)
		}
	})

	// 等待完成
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n=== 测试完成 ===")
}
