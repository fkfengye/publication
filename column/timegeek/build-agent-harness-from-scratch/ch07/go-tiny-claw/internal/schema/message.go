/*
Go 语法速查:

── 类型定义 ──
  type Role string
    说明  基于已有类型定义新类型，称为"类型别名"(或称自定义类型)
    用法  type Role string 定义了一个全新类型 Role，底层存储仍是 string。
          即使底层类型相同，Role 和 string 也不能直接赋值或混用，
          需要显式类型转换：string(Role("user"))。这提供了编译期类型安全

── 常量定义 ──
  const ( ... )
    说明  使用 iota 或直接赋值定义一组常量
    用法  const ( RoleSystem Role = "system" ) — 显式指定类型和值。
          Go 的 const 可以是无类型常量（数字、字符串、布尔），
          也可以像这样带类型声明

── 结构体 ──
  type StructName struct { ... }
    说明  定义结构体类型，Go 中组织相关数据的主要方式
    用法  type Message struct { Role Role; Content string }

  结构体字段标签
    说明  字段后的反引号内容是元数据，用于序列化/反序列化控制
    用法  `json:"role"` — 告诉 encoding/json 包序列化时字段名为 "role"
          而非 "Role"。omitempty 选项表示零值时省略该字段

── 类型组合 ──
  json.RawMessage
    说明  保留原始 JSON 字节片段的类型，延迟解析或透传
    用法  Arguments json.RawMessage — 将 JSON 字符串作为 []byte 存储，
          在使用时再按需 json.Unmarshal。适合需要传递任意结构的数据

── 可见性 ──
  大写首字母 = 导出（公开）
    说明  类型名、字段名、函数名首字母大写表示跨包可见
    用法  type Message struct { Role Role; Content string } —
          Message 和所有字段都是公开的，可以被 internal/schema 包以外访问

── 空接口 ──
  interface{}
    说明  可以接收任意类型的值，类似 Java 的 Object / C# 的 object
    用法  InputSchema interface{} — 在这里可以存放任意 JSON Schema 结构，
          具体类型在运行时通过类型断言（m.(map[string]interface{})）确定
*/

package schema

import "encoding/json"

// Role 表示消息的角色类型（system/user/assistant）
type Role string

const (
	RoleSystem    Role = "system"    // 系统提示词，设定模型行为
	RoleUser      Role = "user"      // 用户输入或工具执行结果的反馈
	RoleAssistant Role = "assistant" // 模型生成的回复（文本或工具调用）
)

// Message 是 Agent 对话中的一条消息，包含角色、内容和可选的工具调用信息
type Message struct {
	Role       Role       `json:"role"`                   // 消息角色
	Content    string     `json:"content"`                // 消息正文文本
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 模型发起的工具调用列表（助理消息）
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具调用 ID，用于关联结果回传
}

// ToolCall 表示模型请求执行的某一个工具调用
type ToolCall struct {
	ID        string          `json:"id"`        // 工具调用唯一标识
	Name      string          `json:"name"`      // 要执行的工具名称
	Arguments json.RawMessage `json:"arguments"` // 工具参数（JSON 格式，待解析）
}

// ToolResult 表示工具执行完成后返回给模型的结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"` // 关联的工具调用 ID
	Output     string `json:"output"`       // 工具执行的输出文本
	IsError    bool   `json:"is_error"`     // 标记执行过程是否发生错误
}

// ToolDefinition 描述一个工具的元信息，用于向模型声明可用工具
type ToolDefinition struct {
	Name        string      `json:"name"`         // 工具名称
	Description string      `json:"description"`  // 工具描述，模型根据此判断何时调用
	InputSchema interface{} `json:"input_schema"` // 参数 JSON Schema（支持任意结构）
}
