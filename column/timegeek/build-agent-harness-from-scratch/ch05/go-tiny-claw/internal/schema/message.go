package schema

import "encoding/json"

/*
Go 语法速查:

── 类型系统 ──
  type Xxx T
    说明  基于底层类型 T 创建新类型，新类型拥有独立的方法集
    用法  type Role string — Role 基于 string，语义更清晰。不能直接当 string 使用，需显式转换

  struct
    说明  字段集合，字段名首字母大写=导出（包外可见），小写=私有（仅包内可见）
    用法  type Message struct { Role Role `json:"role"` } — 字段可加 json tag 控制序列化

  bool
    说明  布尔类型，只有 true 和 false 两个值，零值为 false
    用法  IsError bool `json:"is_error"` — 标记工具执行是否出错

  interface{}
    说明  空接口，可接收任意类型的值（类似 Java Object）
    用法  InputSchema interface{} — 允许字段接受任意 JSON 结构。Go 1.18+ 推荐用 any 替代

  const
    说明  常量声明，值在编译期确定
    用法  const ( RoleSystem Role = "system" ) — 用 const () 批量定义枚举常量

── 复合类型 ──
  []Type
    说明  切片，长度可变的动态数组
    用法  ToolCalls []ToolCall — 声明 ToolCall 切片字段；用 append 追加，for range 遍历

  json.RawMessage
    说明  保留原始 JSON 字节不解析，底层是 []byte，允许延迟解码
    用法  Arguments json.RawMessage — 适合参数格式不确定的场景，由后续代码按需解析

── 格式化 ──
  json tag
    说明  写在 struct 字段的反引号中，控制 JSON 序列化/反序列化行为
    用法  `json:"role"` 指定字段名为 role；`json:"tool_calls,omitempty"` 中 omitempty
          表示字段为零值时输出 JSON 时省略该字段
*/

// 定义 Role 类型，用于标识消息的发送者角色
type Role string

// 角色枚举常量
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// 定义消息结构体，包含角色、内容、工具调用列表、工具调用关联 ID
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// 定义工具调用结构体，包含调用 ID、工具名、参数原始 JSON
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// 定义工具执行结果结构体
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error"`
}

// 定义工具定义结构体，描述工具的名称、说明和参数 schema
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}
