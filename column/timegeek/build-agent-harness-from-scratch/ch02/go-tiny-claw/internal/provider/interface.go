package provider

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

interface 接口类型: 定义方法签名，实现是隐式的（无需 implements 关键字）
func 方法签名: func Name(params) returns，多返回值用逗号分隔
*Type 指针类型: &取地址，*解引用，指针避免大对象拷贝
context.Context 请求上下文，Go 惯例放在第一个参数
[]Type 切片类型
包名.类型名 访问其他包的导出类型
包名.导出常量 访问其他包的导出常量
*/

// 定义 LLMProvider 接口，约束大模型交互必须实现的方法
type LLMProvider interface {
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}