/*
Go 语法速查:

── 结构体 ──
  type StructName struct { field type }
    说明  定义结构体，可以包含不同类型字段
    用法  type AgentEngine struct { provider provider.LLMProvider; registry tools.Registry }
          结构体字段也遵循首字母大写的导出规则：WorkDir 外部可访问，provider 包私有

  结构体"构造"函数
    说明  Go 没有类/构造函数关键字，惯例用 NewXxx 函数创建实例
    用法  func NewAgentEngine(...) *AgentEngine { return &AgentEngine{...} } —
          返回指针避免拷贝整个结构体。Go 中推荐返回指针而非值

── 方法 ──
  func (e *AgentEngine) Run(...) error
    说明  为结构体定义方法，*AgentEngine 是接收者类型（指针接收者）
    用法  指针接收者可以修改结构体字段。若方法不需要修改字段可用值接收者

── 循环 ──
  for { ... }
    说明  无条件 for 相当于 while(true)，无限循环
    用法  for { ... } — 内部通过 if len(actionResp.ToolCalls) == 0 { break } 跳出

  for range
    说明  遍历切片/数组/map/通道
    用法  for _, toolCall := range actionResp.ToolCalls { ... } —
          下划线 _ 忽略索引，Go 中未使用的变量会编译报错

── 切片 ──
  append(slice, element)
    说明  向切片追加元素，容量不足时自动扩容
    用法  contextHistory = append(contextHistory, *actionResp) —
          扩容量通常为原长度的 2 倍。append 返回新切片，必须重新赋值

── 错误处理 ──
  %w
    说明  fmt.Errorf 中的 %w 动词用于包装错误，支持 errors.Is/errors.As 链式判断
    用法  return fmt.Errorf("Action 阶段生成失败: %w", err) —
          %w 只能使用一次，且只能用于 error 类型的值

  log.Printf
    说明  格式化日志输出，不会终止程序
    用法  log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(actionResp.ToolCalls)) —
          相比 fmt.Printf 多了时间戳前缀

── 指针和解引用 ──
  *struct
    说明  指针类型，自动解引用访问字段
    用法  actionResp.Content — 即使 actionResp 是 *Message，Go 也允许直接 . 访问字段

── 接口 ──
  接口类型作为字段
    说明  结构体字段可以是接口类型，实现依赖注入
    用法  provider provider.LLMProvider — AgentEngine 不关心具体实现，
          只要实现了 LLMProvider 接口就可以注入
*/

package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/schema"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

// AgentEngine 是 Agent 的主引擎，管理 LLM Provider、工具注册表和执行循环
type AgentEngine struct {
	provider       provider.LLMProvider // LLM 调用接口
	registry       tools.Registry       // 工具注册表
	WorkDir        string               // 工作目录（公开字段，外部可访问）
	EnableThinking bool                 // 是否启用慢思考模式（Thinking Phase）
}

// NewAgentEngine 创建 Agent 引擎实例，注入 provider、registry 和配置
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// Run 启动 Agent 主循环：组装消息历史 → [Thinking Phase] → Action Phase → 执行工具 → 循环
func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式 (Thinking Phase): %v\n", e.EnableThinking)

	// 初始化对话上下文：包含系统提示词和用户请求
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

	// Agent 主循环：反复执行直到模型不再请求调用工具
	for {
		turnCount++
		log.Printf("\n========== [Turn %d] 开始 ==========\n", turnCount)

		// 获取当前所有可用工具的定义列表
		availableTools := e.registry.GetAvailableTools()

		// ================= Phase 1: Thinking（慢思考阶段）=================
		// 若开启慢思考模式，剥夺工具访问权让模型先思考规划
		if e.EnableThinking {
			log.Println("[Engine][Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")
			// 传入 nil 禁止模型调用工具，强制其仅输出思考过程
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w", err)
			}
			if thinkResp.Content != "" {
				fmt.Printf("🧠 [内部思考 Trace]: %s\n", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		// ================= Phase 2: Action（执行阶段）=================
		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		// 传入可用工具列表，允许模型调用工具
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}

		// 将模型的行动结果追加到对话上下文
		contextHistory = append(contextHistory, *actionResp)

		// 如果模型有文本回复，打印到控制台
		if actionResp.Content != "" {
			fmt.Printf("🤖 [对外回复]: %s\n", actionResp.Content)
		}

		// ================= 执行判断 =================
		// 模型未请求调用任何工具 → 任务完成，退出循环
		if len(actionResp.ToolCalls) == 0 {
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			break
		}

		log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(actionResp.ToolCalls))

		// 逐个执行模型请求的工具调用
		for _, toolCall := range actionResp.ToolCalls {
			log.Printf("  -> 🛠️ 执行工具: %s, 参数: %s\n", toolCall.Name, string(toolCall.Arguments))

			// 调用注册表中的工具执行方法
			result := e.registry.Execute(ctx, toolCall)

			if result.IsError {
				log.Printf("  -> ❌ 工具执行报错: %s\n", result.Output)
			} else {
				log.Printf("  -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
			}

			// 将工具执行结果构造成一条用户角色消息，追加到上下文
			observationMsg := schema.Message{
				Role:       schema.RoleUser,
				Content:    result.Output,
				ToolCallID: toolCall.ID,
			}
			contextHistory = append(contextHistory, observationMsg)
		}
		// 循环继续：下一轮模型看到工具结果后决定下一步行动
	}

	return nil
}
