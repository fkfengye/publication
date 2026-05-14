/*
Go 语法速查:

── 文件操作 ──
  os.Open / file.Close()
    说明  打开文件获取 *os.File，使用后必须关闭
    用法  file, err := os.Open(fullPath); defer file.Close() —
          打开文件后立即 defer 关闭。Go 的 defer 在函数 return 时执行，
          即使中间发生 panic 也会执行

  io.ReadAll
    说明  从 reader 读取所有内容直到 EOF，返回 []byte
    用法  content, err := io.ReadAll(file) — 适合文件大小适中的场景。
          超大文件应使用 bufio 逐行读取

── defer ──
  defer file.Close()
    说明  延迟执行关闭操作，确保资源释放
    用法  在打开文件成功检查后立即 defer，避免忘记关闭导致文件描述符泄漏。
          多个 defer 按后进先出（LIFO）顺序执行

── 路径操作 ──
  filepath.Join
    说明  跨平台路径拼接，自动处理分隔符
    用法  fullPath := filepath.Join(t.workDir, input.Path) —
          Windows 上输出 workDir\input.Path，Unix 上输出 workDir/input.Path

── 错误处理 ──
  *os.PathError
    说明  os.Open / os.ReadFile 在文件不存在时返回路径相关错误
    用法  err 中可通过 errors.Unwrap 或 fmt.Errorf 的 %w 包装后传递
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

// ReadFileTool 提供读取文件内容的能力
type ReadFileTool struct {
	workDir string // 工作目录，所有路径基于此解析
}

// NewReadFileTool 创建读取文件工具实例
func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Definition 返回工具的 JSON Schema 定义，描述 path 参数
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

// readFileArgs 用于解析工具调用中的 JSON 参数
type readFileArgs struct {
	Path string `json:"path"`
}

// Execute 读取指定文件的内容并返回，超长内容自动截断至 8000 字节
func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 解析模型传入的 JSON 参数
	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 拼接完整路径
	fullPath := filepath.Join(t.workDir, input.Path)

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 一次性读取全部内容
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 输出过长时截断至 8000 字节
	const maxLen = 8000
	if len(content) > maxLen {
		truncatedMsg := fmt.Sprintf("%s\n\n...[由于内容过长，已被系统截断至前 %d 字节]...", string(content[:maxLen]), maxLen)
		return truncatedMsg, nil
	}

	return string(content), nil
}
