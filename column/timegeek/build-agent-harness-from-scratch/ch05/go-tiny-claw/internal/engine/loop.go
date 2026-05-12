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

── 类型系统 ──
  type struct
    说明  定义结构体，字段首字母大写=导出，小写=私有
    用法  type AgentEngine struct { provider provider.LLMProvider; EnableThinking bool }

  &Struct{...}
    说明  创建结构体并取地址，返回指针
    用法  return &AgentEngine{provider: p, registry: r, WorkDir: workDir, EnableThinking: enableThinking}

── 函数与方法 ──
  func NewXxx(...)
    说明  构造函数命名惯例，返回 *AgentEngine 指针
    用法  func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine

  (e *Type) Method()
    说明  方法接收者，指针接收者可修改原始对象
    用法  func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error

  多返回值
    说明  多返回值用逗号分隔，func Run(...) error 只返回错误，成功返回 nil
    用法  调用方用 if err := eng.Run(...); err != nil { ... } 检查

── 变量与指针 ──
  :=
    说明  短变量声明，自动推断类型，仅函数内可用
    用法  turnCount := 0 — 等价于 var turnCount int = 0

  nil
    说明  零值，表示指针/接口/切片/map/chan/func 的"空"状态
    用法  e.provider.Generate(ctx, contextHistory, nil) — 传入 nil 剥夺工具（空切片≠nil）

── 复合类型 ──
  []Type{...}
    说明  切片字面量，在声明时直接初始化元素
    用法  contextHistory := []schema.Message{{Role: schema.RoleSystem, Content: "..."}}

── 控制流 ──
  for {}
    说明  无条件的无限循环，必须用 break 或 return 退出
    用法  for { ... break ... } — Agent 主循环，模型不再请求工具调用时 break

  for range {}
    说明  遍历切片，每次迭代返回索引和元素值（值拷贝）
    用法  for _, toolCall := range actionResp.ToolCalls — _ 丢弃索引，只取元素

  break
    说明  跳出当前最内层循环
    用法  if len(actionResp.ToolCalls) == 0 { break } — 无工具调用时退出循环

  if err != nil
    说明  Go 错误处理标准惯用模式
    用法  resp, err := e.provider.Generate(...); if err != nil { return fmt.Errorf("...: %w", err) }

── 错误处理 ──
  fmt.Errorf + %w
    说明  创建带格式的错误消息，%w 包装原始错误保留错误链
    用法  return fmt.Errorf("Thinking 阶段生成失败: %w", err)

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 引用其他包导出的类型
    用法  provider.LLMProvider, schema.Message, tools.Registry

── 内置函数 ──
  len()
    说明  返回集合长度
    用法  len(actionResp.ToolCalls) — 判断切片是否为空（nil 切片 len 也返回 0）

  append()
    说明  向切片末尾追加元素，可能触发扩容分配新底层数组
    用法  contextHistory = append(contextHistory, *actionResp) — 必须用变量接收返回值

  *ptr
    说明  解引用指针，取出指针指向的值
    用法  contextHistory = append(contextHistory, *actionResp) — actionResp 是指针，*取值后追加

── 格式化 ──
  log.Printf
    说明  格式化日志输出，支持 %s %d %v 等占位符
    用法  log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)

  fmt.Printf
    说明  格式化输出到标准输出（控制台），不同于 log.Printf 带时间戳
    用法  fmt.Printf("[对外回复]: %s\n", actionResp.Content)
*/

// 定义 AgentEngine 结构体，包含 LLM Provider、工具注册表、工作区路径和 Thinking 开关
type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	WorkDir        string
	EnableThinking bool
}

// 构造函数，初始化 Agent 引擎
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// 启动 Agent 主循环，支持双阶段（Thinking + Action）执行模式
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

	// Agent 主循环：Thinking → Action → 执行工具 → 重复
	for {
		turnCount++
		log.Printf("\n========== [Turn %d] 开始 ==========\n", turnCount)

		availableTools := e.registry.GetAvailableTools()

		// ================= Phase 1: Thinking（慢思考规划阶段）=================
		if e.EnableThinking {
			log.Println("[Engine][Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")
			// 传入 nil 而非空切片 — nil 表示不提供工具，空切片表示提供但为空
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w", err)
			}
			// 将思考结果追加到对话历史（模型自己看到），不展示给用户
			if thinkResp.Content != "" {
				fmt.Printf("[内部思考 Trace]: %s\n", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		// ================= Phase 2: Action（工具调用执行阶段）=================
		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}

		// 将模型回复追加到对话历史
		contextHistory = append(contextHistory, *actionResp)

		// 输出模型对外回复
		if actionResp.Content != "" {
			fmt.Printf("[对外回复]: %s\n", actionResp.Content)
		}

		// ================= 执行判断 =================
		// 如果没有工具调用请求，表示任务完成，退出主循环
		if len(actionResp.ToolCalls) == 0 {
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			break
		}

		log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(actionResp.ToolCalls))

		// 遍历每个工具调用并执行
		for _, toolCall := range actionResp.ToolCalls {
			log.Printf("  -> 执行工具: %s, 参数: %s\n", toolCall.Name, string(toolCall.Arguments))

			result := e.registry.Execute(ctx, toolCall)

			if result.IsError {
				log.Printf("  -> 工具执行报错: %s\n", result.Output)
			} else {
				log.Printf("  -> 工具执行成功 (返回 %d 字节)\n", len(result.Output))
			}

			// 将工具执行结果作为用户消息追加到对话历史，供下一轮 LLM 使用
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
