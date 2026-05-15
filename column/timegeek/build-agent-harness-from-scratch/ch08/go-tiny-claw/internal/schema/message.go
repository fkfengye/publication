/*
Go 语法速查:

── 类型定义 ──
  type Xxx T
    说明  基于底层类型 T 创建新类型，拥有独立的方法集
    用法  type Role string — Role 是独立类型，不能直接与 string 运算，需显式转换

── 常量组 ──
  const ( ... )
    说明  定义一组相关常量，使用圆括号分组
    用法  const ( RoleSystem Role = "system"; RoleUser Role = "user" )

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出（包外可见），小写=私有（仅包内可见）
    用法  type Message struct { Role Role; Content string }，
          字段可加 json tag 控制序列化：`json:"role"`

── JSON Tag ──
  `json:"field_name,omitempty`
    说明  控制结构体字段序列化时的 JSON 键名，omitempty 表示零值时忽略该字段
    用法  `json:"content"`, `json:"tool_calls,omitempty"`（切片为 nil 时不输出）

── 切片字段 ──
  []Type
    说明  切片，底层是动态数组。长度可变，是 Go 中最常用的集合类型
    用法  ToolCalls []ToolCall，初始化为 nil，append 时自动扩容

── JSON 处理 ──
  json.RawMessage
    说明  原始 JSON 字节数组，不做解析，用于存储未知结构的 JSON 数据
    用法  Arguments json.RawMessage，保留原始参数供下游工具解析
*/

package schema

import "encoding/json"

// Role 表示对话中消息发送者的角色
type Role string

// 预定义的角色常量
const (
	RoleSystem    Role = "system"    // 系统角色，用于设置 Agent 行为规范
	RoleUser      Role = "user"      // 用户角色，人类发送的消息
	RoleAssistant Role = "assistant" // 助手角色，AI 生成的消息
)

// Message 表示对话中的一条消息
type Message struct {
	Role       Role       `json:"role"`                   // 消息发送者角色
	Content    string     `json:"content"`                // 消息文本内容
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 工具调用请求列表（可选）
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具调用 ID，用于关联工具结果
}

// ToolCall 表示模型请求调用工具的意图
type ToolCall struct {
	ID        string          `json:"id"`        // 工具调用的唯一标识
	Name      string          `json:"name"`      // 被调用工具的名称
	Arguments json.RawMessage `json:"arguments"` // 工具调用的参数（JSON 格式）
}

// ToolResult 表示工具执行的结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"` // 对应的工具调用 ID
	Output     string `json:"output"`       // 工具执行结果（文本）
	IsError    bool   `json:"is_error"`     // 执行是否出错
}

// ToolDefinition 描述工具的能力和参数规范，供模型了解如何使用工具
type ToolDefinition struct {
	Name        string      `json:"name"`         // 工具名称
	Description string      `json:"description"`  // 工具功能描述
	InputSchema interface{} `json:"input_schema"` // 工具输入参数规范（JSON Schema 格式）
}
