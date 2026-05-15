/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type WriteFileTool struct { workDir string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针
    用法  func NewWriteFileTool(workDir string) *WriteFileTool { return &WriteFileTool{...} }

── 方法接收者 ──
  (t *WriteFileTool) Method()
    说明  指针接收者方法
    用法  func (t *WriteFileTool) Name() string { return "write_file" }

── JSON Tag ──
  `json:"field_name"`
    说明  控制结构体字段序列化时的 JSON 键名
    用法  Path string `json:"path"`; Content string `json:"content"`

── 文件路径 ──
  filepath.Join / filepath.Dir
    说明  Join 连接路径片段，Dir 获取父目录路径
    用法  fullPath := filepath.Join(t.workDir, input.Path); dir := filepath.Dir(fullPath)

── 目录创建 ──
  os.MkdirAll
    说明  递归创建目录，如果目录已存在不报错
    用法  os.MkdirAll(dir, 0755)

── 文件写入 ──
  os.WriteFile
    说明  写入内容到文件（原子性），自动创建父目录
    用法  os.WriteFile(fullPath, []byte(input.Content), 0644)
*/

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// WriteFileTool 创建或覆盖写入文件
type WriteFileTool struct {
	workDir string
}

// NewWriteFileTool 构造函数，创建 WriteFileTool 实例
func NewWriteFileTool(workDir string) *WriteFileTool {
	return &WriteFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Definition 返回工具定义，包含工具名称、描述和参数规范
func (t *WriteFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "创建或覆盖写入一个文件。如果目录不存在会自动创建。请提供相对于工作区的相对路径。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要写入的文件路径，如 src/main.go",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "要写入的完整文件内容",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// writeFileArgs 定义 write_file 工具的输入参数结构
type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Execute 将内容写入指定文件
func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input writeFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	fullPath := filepath.Join(t.workDir, input.Path)

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("创建父目录失败: %w", err)
	}

	err := os.WriteFile(fullPath, []byte(input.Content), 0644)
	if err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("成功将内容写入到文件: %s", input.Path), nil
}
