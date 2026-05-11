package schema

import "encoding/json"

/*
Go 语法速查:

type 定义类型，type Xxx T 是基于底层类型创建新类型
struct 结构体，字段名首字母大写 = 导出，小写 = 私有
json tag: `json:"字段名,omitempty"` 告诉 json 包序列化时的字段名
omitempty: 当字段为零值时序列化时省略该字段
[]Type 切片类型，类似动态数组
interface{} 空接口，可接收任意类型（类似 Java Object）
bool 布尔类型，值: true / false
json.RawMessage 保留原始 JSON 字符串不解析，底层是 []byte
*/
type Role string

// 定义角色枚举常量
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

// 定义工具调用结构体，包含 ID、工具名、参数字符串
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

// 定义工具定义结构体，包含名称、描述、参数 schema
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}