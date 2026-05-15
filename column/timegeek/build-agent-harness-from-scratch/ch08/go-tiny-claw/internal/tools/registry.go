/*
Go 语法速查:

── 接口 ──
  interface
    说明  定义一组方法签名，不包含实现；实现是隐式的
    用法  type Registry interface { Register(tool BaseTool); GetAvailableTools() []ToolDefinition }

── map 类型 ──
  map[K]V
    说明  键值对映射，K 必须是可比较类型（string/int/指针等），V 任意
    用法  tools := make(map[string]BaseTool)，用 m["key"] = value 写入，v, ok := m["key"] 安全读取

── 初始化 ──
  make(map[K]V)
    说明  初始化 map，必须先 make 才能使用，否则为 nil，写入会 panic
    用法  tools: make(map[string]BaseTool)

── 错误处理 ──
  fmt.Errorf
    说明  格式化错误信息，返回 error 接口
    用法  return "", fmt.Errorf("Error: 系统中不存在名为 '%s' 的工具。", call.Name)

── 字符串格式化 ──
  fmt.Sprintf
    说明  格式化字符串并返回，不输出到标准输出
    用法  errMsg := fmt.Sprintf("Error: %s not found", name)
*/

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// BaseTool 定义所有工具必须实现的接口
type BaseTool interface {
	Name() string                                                      // 返回工具名称
	Definition() schema.ToolDefinition                                 // 返回工具定义（名称、描述、参数规范）
	Execute(ctx context.Context, args json.RawMessage) (string, error) // 执行工具逻辑
}

// Registry 是工具的注册中心，负责管理工具的注册、查询和执行
type Registry interface {
	Register(tool BaseTool)                                              // 注册工具
	GetAvailableTools() []schema.ToolDefinition                          // 获取所有可用工具定义
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult // 执行指定的工具调用
}

// registryImpl 是 Registry 接口的具体实现
type registryImpl struct {
	tools map[string]BaseTool // 工具名称到工具实例的映射
}

// NewRegistry 构造函数，创建新的工具注册中心
func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTool),
	}
}

// Register 注册工具到注册中心，同名工具会被覆盖
func (r *registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[Warning] 工具 '%s' 已经被注册，将被覆盖。\n", name)
	}
	r.tools[name] = tool
	log.Printf("[Registry] 成功挂载工具: %s\n", name)
}

// GetAvailableTools 返回所有已注册工具的定义，供 LLM 了解可用工具
func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var defs []schema.ToolDefinition
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

// Execute 根据工具调用请求执行对应的工具
func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	tool, exists := r.tools[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	output, err := tool.Execute(ctx, call.Arguments)

	if err != nil {
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Error executing %s: %v", call.Name, err),
			IsError:    true,
		}
	}

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
