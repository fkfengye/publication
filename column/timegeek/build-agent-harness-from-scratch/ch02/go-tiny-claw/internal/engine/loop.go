package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/schema"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

/*
Go 语法速查:

type struct 定义结构体，字段名大写导出，小写私有
包名.类型名 引用其他包的导出类型作为字段类型
func NewXxx 构造函数命名惯例
&Struct{...} 创建结构体并取地址，返回指针
(e *Type) 方法接收者，将函数绑定到类型上，指针接收者可修改原始对象
:= 短变量声明，自动推断类型
[]Type{...} 切片字面量
for {} 无限循环，for range {} 遍历切片
break 跳出循环
if err != nil 错误检查模式
return 多返回值，return nil 表示无错误
append 内置函数向切片追加元素
*ptr 解引用指针获取值
len() 内置函数返回长度
fmt.Errorf + %w 错误包装
log.Printf/Println/Log.Fatalf 日志输出
%占位符: %s字符串 %d整数 %v通用
*/

// 定义 AgentEngine 结构体，包含大模型提供者、工具注册表、工作目录
type AgentEngine struct {
	provider provider.LLMProvider
	registry tools.Registry
	WorkDir  string
}

// 构造函数，初始化 AgentEngine
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string) *AgentEngine {
	return &AgentEngine{
		provider: p,
		registry: r,
		WorkDir:  workDir,
	}
}

// 启动 Agent 主循环，接收用户任务描述并执行
func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)

	// 初始化对话历史，包含系统提示和用户输入
	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}

	turnCount := 0

	// Agent 主循环：不断调用大模型，直到任务完成
	for {
		turnCount++
		log.Printf("========== [Turn %d] 开始 ==========\n", turnCount)

		// 获取可用工具列表
		availableTools := e.registry.GetAvailableTools()

		log.Println("[Engine] 正在思考 (Reasoning)...")

		// 调用大模型生成回复
		responseMsg, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("模型生成失败: %w", err)
		}

		// 将模型回复追加到对话历史
		contextHistory = append(contextHistory, *responseMsg)

		// 输出模型回复内容
		if responseMsg.Content != "" {
			fmt.Printf("🤖 模型: %s\n", responseMsg.Content)
		}

		// 如果没有工具调用，表示任务完成，退出循环
		if len(responseMsg.ToolCalls) == 0 {
			log.Println("[Engine] 任务完成，退出循环。")
			break
		}

		log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(responseMsg.ToolCalls))

		// 遍历并执行每个工具调用
		for _, toolCall := range responseMsg.ToolCalls {
			log.Printf("  -> 🛠️ 执行工具: %s, 参数: %s\n", toolCall.Name, string(toolCall.Arguments))

			// 执行工具并获取结果
			result := e.registry.Execute(ctx, toolCall)

			// 根据执行结果输出日志
			if result.IsError {
				log.Printf("  -> ❌ 工具执行报错: %s\n", result.Output)
			} else {
				log.Printf("  -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
			}

			// 将工具执行结果作为用户消息追加到对话历史，供下一轮使用
			observationMsg := schema.Message{
				Role:       schema.RoleUser,
				Content:    result.Output,
				ToolCallID: toolCall.ID,
			}
			contextHistory = append(contextHistory, observationMsg)
		}
	}

	return nil
}