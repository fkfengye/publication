package schema

import "encoding/json"

/*
Go 语法速查:

── 类型系统 ──
  type Xxx T
    说明  基于底层类型 T 创建新类型，新类型拥有独立的方法集
    用法  type Role string — Role 是基于 string 的新类型，不能直接当 string 使用。
          常用于给基础类型附加语义，让代码更清晰

  struct
    说明  字段集合，字段名首字母大写=导出（包外可见），小写=私有（仅包内可见）
    用法  type Message struct { Role Role `json:"role"` }。构造 Message{Role: "user"}，
          未赋值的字段为对应类型的零值

  bool
    说明  布尔类型，只有两个值：true 和 false
    用法  var isError bool = true；bool 类型的零值是 false。常用于标记状态、判断条件

  interface{}
    说明  空接口，可接收任意类型的值（类似 Java Object 或 C void*）
    用法  InputSchema interface{} 允许字段接受任何类型的值。读取后需要用类型断言
          v, ok := x.(string) 取出具体类型。Go 1.18+ 推荐用 any 替代 interface{}

── 复合类型 ──
  []Type
    说明  切片，长度可变的动态数组，是 Go 中最常用的集合类型
    用法  ToolCalls []ToolCall 声明一个 ToolCall 切片字段。用 make 或字面量初始化，
          用 append 追加元素，用 for range 遍历

  json.RawMessage
    说明  保留原始 JSON 字节不解析，允许延迟解码
    用法  Arguments json.RawMessage `json:"arguments"` — 底层是 []byte，存储 JSON 原始文本。
          适合参数内容不确定的场景，由后续代码按需解析

── 格式化 ──
  json tag
    说明  写在 struct 字段的反引号中，控制 JSON 序列化和反序列化行为
    用法  `json:"role"` 指定 JSON 字段名为 role；`json:"tool_calls,omitempty"` 中 omitempty
          表示字段为零值时输出 JSON 时省略该字段，减少冗余数据

  omitempty
    说明  json tag 的可选标记，零值时序列化省略该字段
    用法  ToolCalls 为 nil 或空切片时，输出 JSON 不包含 tool_calls 字段。
          注意 bool 零值 false 也会触发省略，用 *bool 可规避
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
