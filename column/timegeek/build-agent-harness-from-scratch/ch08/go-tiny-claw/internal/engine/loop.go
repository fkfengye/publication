/*
Go 语法速查:

── 结构体与方法 ──
  type Struct struct { ... }
    说明  定义结构体，字段集合。首字母大写=导出（包外可见），小写=私有（仅包内可见）
    用法  type AgentEngine struct { provider LLMProvider }，用 &Struct{} 或 Struct{} 创建实例

  (e *AgentEngine) Method()
    说明  指针接收者方法，可访问和修改结构体字段
    用法  func (e *AgentEngine) Run(ctx context.Context, prompt string) error { ... }

── 接口 ──
  接口类型作为字段
    说明  结构体中可包含接口类型字段，实现依赖注入和多态
    用法  provider LLMProvider — 运行时注入不同的 LLM 实现（OpenAI/Claude）

── 切片字面量 ──
  []Type{ ... }
    说明  切片字面量创建并初始化切片，底层数组直接存储元素
    用法  contextHistory := []schema.Message{{Role: schema.RoleSystem, Content: "..."}}

── for 循环 ──
  for { ... }
    说明  无限循环，配合 break/return 退出，或 continue 跳过本次迭代
    用法  for { if condition { break } }，Agent 主循环持续运行直到任务完成

── 并发 ──
  sync.WaitGroup
    说明  计数器锁，用于等待一组 goroutine 完成。Add() 增加计数，Done() 减一，Wait() 阻塞直到归零
    用法  var wg sync.WaitGroup; wg.Add(1); go func(){ defer wg.Done(); ... }(); wg.Wait()

  go func() { ... }(args)
    说明  启动新的 goroutine（轻量级线程），匿名函数立即执行
    用法  go func(idx int, call schema.ToolCall) { ... }(i, toolCall)

── 切片预分配 ──
  make([]Type, len)
    说明  预分配指定长度的切片，元素为零值，可避免扩容和并发写入时的数据竞争
    用法  observationMsgs := make([]schema.Message, len(toolCalls))，按索引写入无需加锁

── 上下文 ──
  context.Context
    说明  携带截止时间、取消信号和请求级别值的上下文，作为第一个参数在函数间传递
    用法  ctx := context.Background() 创建，WithTimeout/WithCancel 派生后传递

── defer ──
  defer func() { ... }()
    说明  延迟执行，在函数返回前按 LIFO 顺序执行，常用于资源释放
    用法  defer wg.Done()，确保每个 goroutine 完成后计数器减一
*/

package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/schema"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

// AgentEngine 是 Agent 的核心执行引擎，管理 LLM Provider、工具注册表和工作区
type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	WorkDir        string
	EnableThinking bool
}

// NewAgentEngine 构造函数，初始化 AgentEngine 实例
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// Run 是 Agent 的主循环，接收用户提示词并执行多轮对话直到任务完成
func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式 (Thinking Phase): %v\n", e.EnableThinking)

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

	// Agent 主循环：持续执行直到模型不再请求工具调用
	for {
		turnCount++
		log.Printf("\n========== [Turn %d] 开始 ==========\n", turnCount)

		availableTools := e.registry.GetAvailableTools()

		// Phase 1: 慢思考阶段（可选）
		// 当启用时，模型先在无工具的情况下进行推理和规划
		if e.EnableThinking {
			log.Println("[Engine][Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w", err)
			}
			if thinkResp.Content != "" {
				fmt.Printf("🧠 [内部思考 Trace]: \n%s\n", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		// Phase 2: 行动阶段
		// 模型根据上下文和可用工具生成回复或工具调用
		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}

		contextHistory = append(contextHistory, *actionResp)

		if actionResp.Content != "" {
			fmt.Printf("🤖 [对外回复]: \n%s\n", actionResp.Content)
		}

		// 如果没有工具调用，表示任务完成，退出循环
		if len(actionResp.ToolCalls) == 0 {
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			break
		}

		log.Printf("[Engine] 模型请求并发调用 %d 个工具...\n", len(actionResp.ToolCalls))

		// 预分配切片以保证顺序并避免并发写入锁
		observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))
		var wg sync.WaitGroup

		// 并发执行所有工具调用
		for i, toolCall := range actionResp.ToolCalls {
			wg.Add(1)

			go func(idx int, call schema.ToolCall) {
				defer wg.Done()

				log.Printf("  -> [Go-%d] 🛠️ 触发并行执行: %s\n", idx, call.Name)

				// 执行底层工具
				result := e.registry.Execute(ctx, call)

				if result.IsError {
					log.Printf("  -> [Go-%d] ❌ 工具执行报错: %s\n", idx, result.Output)
				} else {
					log.Printf("  -> [Go-%d] ✅ 工具执行成功 (返回 %d 字节)\n", idx, len(result.Output))
				}

				// 安全写入对应索引
				observationMsgs[idx] = schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: call.ID,
				}
			}(i, toolCall)
		}

		wg.Wait() // 阻塞等待所有 goroutine 完成
		log.Println("[Engine] 所有并发工具执行完毕，开始聚合观察结果 (Observation)...")

		// 按序追加回 Context，为下一轮对话做准备
		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
	}

	return nil
}
