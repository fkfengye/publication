package tools

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

interface 接口: 定义一组方法签名，实现类是隐式的
func 方法定义: 参数名在前类型在后（与 Java 相反）
() 空参数列表
多返回值: returns 中用逗号分隔多个类型
值返回 vs 指针返回: 小结构体通常值返回，拷贝开销可忽略
*/

// 定义 Registry 接口，约束工具注册表必须实现的方法
type Registry interface {
	// 获取可用工具列表
	GetAvailableTools() []schema.ToolDefinition
	// 执行工具调用并返回结果
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}