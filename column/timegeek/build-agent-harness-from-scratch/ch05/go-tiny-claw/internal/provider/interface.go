package provider

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  interface
    说明  定义一组方法签名，实现是隐式的，无需 implements 关键字
    用法  用 type Xxx interface { ... } 声明；任意类型只要实现了接口的所有方法就自动满足该接口

── 函数与方法 ──
  func Name(p T) Ret
    说明  函数签名，参数名在前类型在后
    用法  func Generate(ctx context.Context, messages []schema.Message, ...) (*schema.Message, error)

  多返回值
    说明  Go 原生支持多返回值，常见模式是 (result, error)
    用法  (*schema.Message, error) — 返回消息指针和错误，调用方必须检查 error

── 变量与指针 ──
  *Type
    说明  指针类型，& 取地址，* 解引用。返回指针避免大对象拷贝，允许返回 nil
    用法  *schema.Message — 返回消息指针，调用方通过 msg.Content 访问字段

── 复合类型 ──
  []Type
    说明  切片，长度可变的动态数组
    用法  []schema.Message, []schema.ToolDefinition — 函数参数中的切片类型

── 上下文 ──
  context.Context
    说明  携带超时、取消信号和请求级别值的上下文，Go 惯例作为函数第一个参数
    用法  从调用方传入，向下传递；不要存入 struct 中

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 访问其他包导出的标识符，首字母大写的才是导出的
    用法  schema.Message — schema 是包名，Message 是导出的结构体
*/

// 定义 LLMProvider 接口，约束大模型交互必须实现的方法
type LLMProvider interface {
	// Generate 接收对话历史和可用工具列表，返回模型回复
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
