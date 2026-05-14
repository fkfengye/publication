/*
Go 语法速查:

── 结构体与方法 ──
  type BashTool struct { workDir string }
    说明  定义包含状态的工具结构体
    用法  每个工具结构体持有所需上下文字段（如工作目录），通过 NewXxx 函数注入

  方法定义
    说明  为结构体实现接口方法
    用法  func (t *BashTool) Name() string { return "bash" } —
          指针接收者。虽然本方法不修改字段，但为保持一致性，所有工具方法都使用指针接收者

── 内嵌结构体 ──
  type bashArgs struct { Command string `json:"command"` }
    说明  用于 JSON 反序列化的参数结构体
    用法  私有类型 bashArgs 只在本文件中使用。字段标签 `json:"command"` 指明 JSON 字段名，
          json.Unmarshal 根据此标签映射字段

── 函数作为方法 ──
  context.WithTimeout
    说明  从父 context 派生一个带超时控制的子 context
    用法  timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second) —
          timeoutCtx 将在 30 秒后或 cancel() 调用时自动取消。
          defer cancel() 确保函数返回时释放资源

── defer ──
  defer cancel()
    说明  延迟执行，函数返回前调用
    用法  defer cancel() — 在创建带超时的 context 后立即 defer 取消，
          防止 context 泄漏

── exec.CommandContext ──
    说明  执行外部命令，可被 context 取消（杀死进程）
    用法  cmd := exec.CommandContext(timeoutCtx, "bash", "-c", input.Command) —
          如果 timeoutCtx 超时，Go 会向子进程发送 Kill 信号

── context.DeadlineExceeded ──
    说明  context 超时后的典型错误
    用法  if timeoutCtx.Err() == context.DeadlineExceeded { ... } —
          用于判断命令是否因超时被终止

── 常量 ──
  const maxLen = 8000
    说明  声明不可变值
    用法  const maxLen = 8000 — Go 的 const 只能是编译期可确定的常量，
          不能是函数返回值。可以声明为无类型常量，使用时自动推断类型

── 切片截取 ──
  outputStr[:maxLen]
    说明  取切片前 maxLen 个元素，左闭右开
    用法  outputStr[:maxLen] — 取前 maxLen 字节。若 outputStr 长度不足 maxLen，
          会 panic，因此先判断 len(outputStr) > maxLen
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

// BashTool 提供在指定工作目录执行 bash 命令的能力
type BashTool struct {
	workDir string // 命令执行的工作目录
}

// NewBashTool 创建 Bash 工具实例
func NewBashTool(workDir string) *BashTool {
	return &BashTool{workDir: workDir}
}

// Name 返回工具名称，供模型在 tool_use 中引用
func (t *BashTool) Name() string {
	return "bash"
}

// Definition 返回工具的 JSON Schema 定义，描述参数结构供模型理解
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

// bashArgs 用于解析工具调用中的 JSON 参数
type bashArgs struct {
	Command string `json:"command"`
}

// Execute 执行 bash 命令并返回输出，包含超时保护、输出截断等安全措施
func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 解析模型传入的 JSON 参数
	var input bashArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 创建带 30 秒超时的子 context，确保命令不会无限执行
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 执行 bash 命令
	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", input.Command)
	cmd.Dir = t.workDir // 设置工作目录

	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	// 检查是否因超时被杀死
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return outputStr + "\n[警告: 命令执行超时(30s)，已被系统强制终止。]", nil
	}

	// 命令执行出错（非零退出码）
	if err != nil {
		return fmt.Sprintf("执行报错: %v\n输出:\n%s", err, outputStr), nil
	}

	// 空输出处理
	if outputStr == "" {
		return "命令执行成功，无终端输出。", nil
	}

	// 输出过长时截断至 8000 字节
	const maxLen = 8000
	if len(outputStr) > maxLen {
		return fmt.Sprintf("%s\n\n...[终端输出过长，已截断至前 %d 字节]...", outputStr[:maxLen], maxLen), nil
	}

	return outputStr, nil
}
