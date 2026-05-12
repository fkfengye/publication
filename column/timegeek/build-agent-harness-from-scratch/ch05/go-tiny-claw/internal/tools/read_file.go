package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  type struct
    说明  定义结构体，小写类型名 = 包私有
    用法  type ReadFileTool struct { workDir string } — workDir 小写，包外不可直接访问

── 函数与方法 ──
  func NewXxx(...)
    说明  Go 无构造函数语法，约定用 New+类型名 的工厂函数返回初始化好的实例
    用法  func NewReadFileTool(workDir string) *ReadFileTool — 返回指针给外部使用

  (t *Type) Method()
    说明  给类型绑定方法，t 为接收者。指针接收者可修改原始对象
    用法  func (t *ReadFileTool) Name() string — t 是 *ReadFileTool 指针

── 复合类型 ──
  map[string]interface{}
    说明  string 到任意类型的映射，用于表示 JSON object
    用法  map[string]interface{}{"type": "object", "properties": map[string]interface{}{...}}
          — 嵌套 map 构造 JSON Schema 结构，interface{} 允许值类型灵活

  type Xxx struct { ... }
    说明  定义参数结构体，配合 json tag 用于反序列化工具调用参数
    用法  type readFileArgs struct { Path string `json:"path"` } — json.Unmarshal 按 tag 匹配

── 错误处理 ──
  json.Unmarshal
    说明  将 JSON 字节反序列化为 Go 结构体或 map
    用法  json.Unmarshal(args, &input) — 第二个参数必须传指针，&input 取 args 结构体地址

  fmt.Errorf + %w
    说明  创建带格式的错误，%w 包装原始错误保留错误链
    用法  fmt.Errorf("参数解析失败: %w", err) — 上层可用 errors.Is/As 追溯原始错误

── 文件操作 ──
  filepath.Join
    说明  跨平台路径拼接，自动使用 OS 正确的分隔符（Windows \，Linux /）
    用法  filepath.Join(t.workDir, input.Path) — 拼接工作区根路径和相对路径

  os.Open
    说明  以只读方式打开文件，返回 *os.File 和 error
    用法  os.Open(fullPath) — 打开的 file 必须用 file.Close() 关闭释放资源

  defer
    说明  延迟执行，函数返回前（无论正常返回还是 panic）按后进先出顺序执行
    用法  defer file.Close() — 紧跟在 os.Open 后，确保文件句柄一定被释放

  io.ReadAll
    说明  一次性读取 io.Reader 的全部内容到 []byte
    用法  io.ReadAll(file) — file 实现了 io.Reader 接口。大文件慎用，全部加载到内存

── 控制流 ──
  if err != nil
    说明  Go 错误检查标准惯用写法
    用法  content, err := io.ReadAll(file); if err != nil { return "", fmt.Errorf(...) }

  const
    说明  编译期常量，值不可变
    用法  const maxLen = 8000 — Go 根据字面量自动推断类型为无类型整数

  string([]byte)
    说明  []byte 到 string 的显式类型转换，底层发生拷贝
    用法  string(content[:maxLen]) — 截取前 maxLen 字节并转为字符串
*/

// 定义 ReadFileTool 结构体，存储工作区根路径
type ReadFileTool struct {
	workDir string
}

// 构造函数，初始化文件读取工具
func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

// 返回工具唯一名称，供 LLM 在 ToolCall 中引用
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// 返回工具定义，包含名称、描述和 JSON Schema 格式的输入参数
func (t *ReadFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "读取指定路径的文件内容。请提供相对工作区的路径。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要读取的文件路径，如 cmd/claw/main.go",
				},
			},
			"required": []string{"path"},
		},
	}
}

// 参数解析用的结构体，定义 read_file 工具接收的 JSON 参数
type readFileArgs struct {
	Path string `json:"path"`
}

// 执行文件读取：解析参数 → 拼接路径 → 打开文件 → 读取内容 → 按需截断
func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 将 LLM 传入的 JSON 参数解析为 readFileArgs 结构体
	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 拼接工作区根路径和相对路径，防止路径穿越
	fullPath := filepath.Join(t.workDir, input.Path)

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 一次性读取文件全部内容到内存
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 内容过长时截断，避免超出 LLM Token 限制
	const maxLen = 8000
	if len(content) > maxLen {
		truncatedMsg := fmt.Sprintf("%s\n\n...[由于内容过长，已被系统截断至前 %d 字节]...", string(content[:maxLen]), maxLen)
		return truncatedMsg, nil
	}

	return string(content), nil
}
