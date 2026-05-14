/*
Go 语法速查:

── 文件操作 ──
  os.MkdirAll
    说明  递归创建目录，类似 mkdir -p
    用法  os.MkdirAll(filepath.Dir(fullPath), 0755) —
          0755 是目录权限：所有者可读写执行，组和其他用户可读执行。
          如果目录已存在，MkdirAll 返回 nil（不报错）

  os.WriteFile
    说明  写入文件，若文件不存在则创建，存在则覆盖
    用法  os.WriteFile(fullPath, []byte(input.Content), 0644) —
          0644 是文件权限。WriteFile 会一次性写入全部内容，适合中小文件

  filepath.Dir
    说明  返回路径中的目录部分
    用法  filepath.Dir("src/main.go") 返回 "src" —
          与 filepath.Join 配合使用确保路径跨平台兼容

── 错误处理 ──
  错误提前返回
    说明  if err != nil { return "", err } — Go 的惯用错误处理模式
    用法  每一步操作后检查 err，有错立即返回，不继续执行
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

// WriteFileTool 提供创建或覆盖写入文件的能力，目录不存在时自动创建
type WriteFileTool struct {
	workDir string // 工作目录，所有路径基于此解析
}

// NewWriteFileTool 创建写入文件工具实例
func NewWriteFileTool(workDir string) *WriteFileTool {
	return &WriteFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Definition 返回工具的 JSON Schema 定义，描述 path 和 content 两个参数
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

// writeFileArgs 用于解析工具调用中的 JSON 参数
type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Execute 写入文件内容，如果父目录不存在则自动创建
func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 解析模型传入的 JSON 参数
	var input writeFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 拼接完整路径
	fullPath := filepath.Join(t.workDir, input.Path)

	// 递归创建父目录（如果目录已存在则无操作）
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("创建父目录失败: %w", err)
	}

	// 写入文件（不存在则创建，存在则覆盖）
	err := os.WriteFile(fullPath, []byte(input.Content), 0644)
	if err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("成功将内容写入到文件: %s", input.Path), nil
}
