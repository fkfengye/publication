package main

import (
	"context"
	"log"
	"os"

	"github.com/yourname/go-tiny-claw/internal/engine"
	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

package main: 可执行程序的入口包，编译器会将其编译成可执行文件
func main: 程序入口点，Go 运行时自动调用，无参无返回值
type struct: 定义结构体
*Type: 指针类型，&取地址创建指针
零值: int=0, string="", bool=false, 指针/接口/切片/chan/func=nil
_ 空白标识符: 丢弃不需要使用的值（参数、返回值）
context.Background(): 创建空根 context，用于程序入口
[]byte(str): 类型转换，字符串转字节切片
`...` 反引号: 原始字符串字面量，内容不转义
struct{}: 空结构体，不占内存
log.Fatalf: 打印日志后 os.Exit(1) 终止程序
*/

// 定义 mockProvider 结构体，实现 LLMProvider 接口
type mockProvider struct {
	turn int
}

// 模拟大模型生成回复，第一轮返回工具调用，第二轮返回纯文本
func (m *mockProvider) Generate(ctx context.Context, msgs []schema.Message, _ []schema.ToolDefinition) (*schema.Message, error) {
	m.turn++
	if m.turn == 1 {
		return &schema.Message{
			Role:    schema.RoleAssistant,
			Content: "让我来看看当前目录下有什么文件。",
			// 返回一个 bash 工具调用，模拟模型请求执行 ls 命令
			ToolCalls: []schema.ToolCall{
				{ID: "call_123", Name: "bash", Arguments: []byte(`{"command": "ls -la"}`)},
			},
		}, nil
	}

	// 第二轮返回纯文本，表示任务完成
	return &schema.Message{
		Role:    schema.RoleAssistant,
		Content: "我看到了文件列表，里面包含 main.go，任务完成！",
	}, nil
}

// 定义 mockRegistry 结构体，实现 Registry 接口
type mockRegistry struct{}

// 返回空工具列表（演示用）
func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition { return nil }

// 模拟执行工具调用，固定返回 ls 输出结果
func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "-rw-r--r--  1 user group  234 Oct 24 10:00 main.go\n",
		IsError:    false,
	}
}

// 程序入口
func main() {
	// 获取当前工作目录
	workDir, _ := os.Getwd()

	// 创建 mock 的 provider 和 registry
	p := &mockProvider{}
	r := &mockRegistry{}

	// 初始化 AgentEngine
	eng := engine.NewAgentEngine(p, r, workDir)

	// 执行 Agent 任务
	err := eng.Run(context.Background(), "帮我检查当前目录的文件")
	if err != nil {
		log.Fatalf("引擎崩溃: %v", err)
	}
}