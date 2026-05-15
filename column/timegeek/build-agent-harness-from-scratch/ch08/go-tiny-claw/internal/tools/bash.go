/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type BashTool struct { workDir string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针，负责初始化结构体
    用法  func NewBashTool(workDir string) *BashTool { return &BashTool{workDir: workDir} }

── 方法接收者 ──
  (t *BashTool) Method()
    说明  指针接收者方法，可访问和修改结构体字段
    用法  func (t *BashTool) Name() string { return "bash" }

── JSON Tag ──
  `json:"field_name"`
    说明  控制结构体字段序列化时的 JSON 键名
    用法  Command string `json:"command"`

── 上下文超时 ──
  context.WithTimeout
    说明  创建带超时限制的子上下文，超时后自动取消
    用法  timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second); defer cancel()

── defer ──
  defer func() { ... }()
    说明  延迟执行，在函数返回前按 LIFO 顺序执行，常用于资源释放和取消函数调用
    用法  defer cancel()，确保函数退出时释放资源

── 命令执行 ──
  exec.CommandContext
    说明  在指定上下文中执行外部命令，上下文可控制超时和取消
    用法  cmd := exec.CommandContext(ctx, "bash", "-c", input.Command)

── 字符串长度截断 ──
  len(string) > maxLen
    说明  获取字符串字节长度，超过限制时截断
    用法  if len(outputStr) > maxLen { return outputStr[:maxLen] + "..." }
*/

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// BashTool 允许在指定工作区执行 bash 命令
type BashTool struct {
	workDir string
}

// NewBashTool 构造函数，创建 BashTool 实例
func NewBashTool(workDir string) *BashTool {
	return &BashTool{workDir: workDir}
}

// Name 返回工具名称
func (t *BashTool) Name() string {
	return "bash"
}

// Definition 返回工具定义，包含工具名称、描述和参数规范
func (t *BashTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "在当前工作区执行任意的 bash 命令。支持链式命令(如 &&)。返回标准输出(stdout)和标准错误(stderr)。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的 bash 命令",
				},
			},
			"required": []string{"command"},
		},
	}
}

// bashArgs 定义 bash 工具的输入参数结构
type bashArgs struct {
	Command string `json:"command"`
}

// Execute 在指定工作区执行 bash 命令，支持超时控制
func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input bashArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 设置 30 秒超时，防止命令长时间阻塞
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", input.Command)
	cmd.Dir = t.workDir

	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	// 检查是否超时
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return outputStr + "\n[警告: 命令执行超时(30s)，已被系统强制终止。]", nil
	}

	if err != nil {
		return fmt.Sprintf("执行报错: %v\n输出:\n%s", err, outputStr), nil
	}

	if outputStr == "" {
		return "命令执行成功，无终端输出。", nil
	}

	// 限制输出长度，避免返回过多内容
	const maxLen = 8000
	if len(outputStr) > maxLen {
		return fmt.Sprintf("%s\n\n...[终端输出过长，已截断至前 %d 字节]...", outputStr[:maxLen], maxLen), nil
	}

	return outputStr, nil
}
