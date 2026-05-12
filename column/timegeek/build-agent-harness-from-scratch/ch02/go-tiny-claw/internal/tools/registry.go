package tools

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  interface
    说明  定义一组方法签名，实现是隐式的，无需 implements 关键字
    用法  用 type Xxx interface { ... } 声明接口。任意类型只要实现了接口的所有方法就自动满足该接口，
          编译器在赋值或传参时检查

── 函数与方法 ──
  func Name(p T) Ret
    说明  函数签名，参数名在前类型在后（与 Java/C 相反）
    用法  func Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult。
          参数列表后写返回值类型，不需要返回值时可以省略

  ()
    说明  空参数列表，函数不接受任何参数时必须写空括号
    用法  GetAvailableTools() — 小括号不能省略，即使没有参数

  值返回 vs 指针返回
    说明  返回值类型：小结构体或不需要修改时直接返回值（会产生拷贝），
          需要修改或避免大对象拷贝时返回指针
    用法  此接口返回 schema.ToolResult（值类型）而非 *schema.ToolResult，因为 ToolResult 结构体
          很小（几个字段），值拷贝开销可忽略
*/

// 定义 Registry 接口，约束工具注册表必须实现的方法
type Registry interface {
	// 获取可用工具列表
	GetAvailableTools() []schema.ToolDefinition
	// 执行工具调用并返回结果
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}
