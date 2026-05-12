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

── 包与导入 ──
  package main
    说明  每个可执行程序必须有一个 main 包，编译器由此生成可执行文件
    用法  package main 声明本文件属于 main 包；一个目录下所有 .go 文件必须属于同一个包

  func main()
    说明  程序入口点，Go 运行时自动调用，无参数无返回值
    用法  每个 main 包必须有且只有一个 func main()。os.Args 获取命令行参数，os.Exit 设置退出码

── 类型系统 ──
  type struct
    说明  定义结构体，字段组合
    用法  type mockProvider struct { turn int } — 结构体将相关字段组织在一起。
          小写 mock 表示该类型为包私有，外部不可见

  struct{}
    说明  无字段的空结构体，占用 0 字节内存
    用法  type mockRegistry struct{} — 适用于只需要方法不需要状态的类型。
          实例化：mockRegistry{}，所有空结构体值都相同

── 变量与指针 ──
  *Type
    说明  指针类型，& 取地址创建指针，* 解引用取值
    用法  p := &mockProvider{} — & 一步完成"构造 + 取地址"。Go 中指针不能运算，
          访问字段直接用 p.Field 自动解引用

  _
    说明  空白标识符，将不需要的值丢弃，避免编译报"变量未使用"
    用法  workDir, _ := os.Getwd() — os.Getwd 返回 (string, error)，这里不关心错误，
          用 _ 接收 error 部分

  零值
    说明  Go 中声明变量不初始化时自动赋予零值，不需要手动赋初始值
    用法  int=0, string="", bool=false, 指针/接口/切片/chan/func=nil。
          mockProvider{} 创建后 turn 字段默认为 0

── 上下文 ──
  context.Background()
    说明  创建空的根 context，通常用于程序入口、测试、初始化阶段
    用法  作为第一个参数传给 Run：eng.Run(context.Background(), "任务描述")。
          正式环境会用 WithTimeout 派生带超时的 context

── 类型转换 ──
  []byte(str)
    说明  将字符串显式转换为字节切片，底层发生拷贝
    用法  []byte(`{"command": "ls -la"}`) 将 JSON 字符串转为字节数组。Go 不支持隐式类型转换，
          所有转换必须显式写出目标类型

── 格式化 ──
  ``（反引号）
    说明  原始字符串字面量，内容不转义，支持跨行
    用法  `{"command": "ls -la"}` 中的双引号不需要转义。适合写 JSON、正则、路径

  log.Fatalf
    说明  格式化输出日志后调用 os.Exit(1) 立即终止程序，defer 语句不会执行
    用法  log.Fatalf("引擎崩溃: %v", err) — %v 自动格式化 err。Fatalf 只在不可恢复的
          致命错误时使用，正常流程用 log.Printf 或 return error
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
