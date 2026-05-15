/*
Go 语法速查:

── 接口 ──
  interface
    说明  定义一组方法签名，不包含实现；实现是隐式的，无需 implements 关键字
    用法  type LLMProvider interface { Generate(...) (Message, error) }。
          任意类型只要实现了接口的所有方法，就自动满足该接口，编译器在赋值时检查

── 上下文 ──
  context.Context
    说明  携带截止时间、取消信号和请求级别值的上下文，Go 惯例作为第一个参数传递
    用法  ctx context.Context，入口处用 context.Background() 创建根 ctx

── 导入 ──
  包名.类型名
    说明  访问其他包的导出类型，只有首字母大写的内容对外可见
    用法  schema.Message — schema 是包名，Message 是导出的结构体
*/

package provider

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// LLMProvider 定义与大语言模型通信的统一接口
type LLMProvider interface {
	// Generate 接收当前对话历史和可用工具列表，返回模型响应
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
