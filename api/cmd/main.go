package main

import (
	"api/router"
	"fmt"
)

type Math struct{ S int }

func main() {
	r := router.NewRouter(nil)
	// 注册函数,使用命名空间
	fmt.Println(r.Register(
		"add",
		"加法计算函数",
		"math",
		[]string{"a", "b"},
		[]string{"c"},
		func(math *Math, a, b int) int {
			panic("死掉了")
			return a + b + math.S
		}))
	// 上下文
	ctx := router.NewContext("miao", "miao", r)
	ctx.SetValue("math", &Math{
		S: 664,
	})
	// 获取函数
	router.CallFunc(ctx, "add", func(fn func(a int, b int) int) {
		fn(1, 1)
	})
}
