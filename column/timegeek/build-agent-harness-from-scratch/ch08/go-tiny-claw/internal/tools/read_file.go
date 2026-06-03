/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type ReadFileTool struct { workDir string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针，负责初始化结构体
    用法  func NewReadFileTool(workDir string) *ReadFileTool { return &ReadFileTool{...} }

── 方法接收者 ──
  (t *ReadFileTool) Method()
    说明  指针接收者方法
    用法  func (t *ReadFileTool) Name() string { return "read_file" }

── JSON Tag ──
  `json:"field_name"`
    说明  控制结构体字段序列化时的 JSON 键名
    用法  Path string `json:"path"`

── 文件路径 ──
  filepath.Join
    说明  连接路径片段，自动处理平台相关的路径分隔符
    用法  fullPath := filepath.Join(t.workDir, input.Path)

── 文件操作 ──
  os.Open / io.ReadAll
    说明  打开文件并读取全部内容
    用法  file, err := os.Open(fullPath); content, err := io.ReadAll(file)

── defer ──
  defer file.Close()
    说明  延迟关闭文件，确保函数退出前释放资源
    用法  defer file.Close()

── 字符串长度截断 ──
  len(string) > maxLen
    说明  获取字符串字节长度
    用法  if len(content) > maxLen { return content[:maxLen] + "..." }
*/

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

// ReadFileTool 读取指定路径的文件内容
type ReadFileTool struct {
	workDir string
}

// NewReadFileTool 构造函数，创建 ReadFileTool 实例
func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Definition 返回工具定义，包含工具名称、描述和参数规范
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

// readFileArgs 定义 read_file 工具的输入参数结构
type readFileArgs struct {
	Path string `json:"path"`
}

// Execute 读取指定文件的内容并返回；自动按 8000 字节截断避免撑爆上下文
func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 1) 反序列化模型传入的 JSON 参数（readFileArgs{Path}）
	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 2) 相对路径 → 绝对路径；所有文件访问都被锁定在 t.workDir 子树内
	fullPath := filepath.Join(t.workDir, input.Path)

	// 3) 只读方式打开文件（O_RDONLY）；不存在的文件直接报错，让模型知道路径有误
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close() // 函数返回前保证关闭，释放文件描述符

	// 4) 一次性把整个文件读入内存（适合代码这种"小到中等"文件）
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 5) 内容超过 8000 字节就截断，附上提示语提醒模型用 edit_file 分段读
	const maxLen = 8000
	if len(content) > maxLen {
		truncatedMsg := fmt.Sprintf("%s\n\n...[由于内容过长，已被系统截断至前 %d 字节]...", string(content[:maxLen]), maxLen)
		return truncatedMsg, nil
	}

	return string(content), nil
}
