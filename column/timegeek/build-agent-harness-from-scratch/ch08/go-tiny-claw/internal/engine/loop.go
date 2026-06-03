/*
Go 语法速查:

【结构体定义】
───────────────────────────────────────────
  type StructName struct { ... }
    说明  定义结构体。结构体是一种复合数据类型，将多个不同类型的字段组合成一个整体。
          字段名首字母大写 = 导出（包外可见），小写 = 私有（仅当前包内可用）。
    用法  type Person struct { Name string; Age int }
          — Person 是类型名，Name 和 Age 是字段

  type FieldType
    说明  基于已有类型创建新类型。新类型有独立的方法集，不能直接和原类型运算。
    用法  type Age int — Age 是新的整数类型
          type Handler func() — Handler 是新的函数类型

【结构体字段类型】
───────────────────────────────────────────
  TypeName        // 接口类型作为字段
    说明  结构体字段可以是任意类型，包括接口类型。接口类型字段必须在构造时赋值（依赖注入）。
    用法  type Engine struct {
            provider LLMProvider  // LLMProvider 是接口，运行时注入具体实现
            registry tools.Registry
          }

【方法声明】
───────────────────────────────────────────
  func (接收者 *TypeName) MethodName(参数) 返回值
    说明  给类型添加方法。接收者类似 Python 的 self 或 Java 的 this。
          指针接收者 (*TypeName) 可以修改原始对象，值接收者 (TypeName) 操作副本。
    用法  func (e *AgentEngine) Run(ctx context.Context, prompt string) error {
            // 在方法内可以通过 e 访问 AgentEngine 的字段
          }
          调用：eng.Run(ctx, prompt) — eng 是 *AgentEngine，指针自动解引用

【切片字面量】
───────────────────────────────────────────
  []Type{value1, value2, ...}
    说明  创建切片并初始化。切片是动态数组，长度可变，底层是数组。
          元素直接放在花括号里，用逗号分隔。
    用法  history := []Message{
            {Role: "system", Content: "你是一个助手"},
            {Role: "user", Content: "你好"}
          }

【for 循环】
───────────────────────────────────────────
  for { 无限循环  }
    说明  没有条件的 for 是无限循环，必须通过 break/return 退出。
          相当于其他语言的 while(true)。
    用法  for {
            if done { break }  // 满足条件时退出循环
            // ...
          }

  for i, v := range slice { .... }
    说明  range 遍历切片或 map，返回索引和值的拷贝（不是引用）。
    用法  for i, msg := range messages {
            // i 是索引，msg 是元素的拷贝
          }

【sync.WaitGroup（并发等待）】
───────────────────────────────────────────
  sync.WaitGroup
    说明  计数器锁，用于等待一组并发任务（goroutine）完成。
          Add(n) 增加计数，Done() 减少计数（调用 Add(1) 后的标准模式），Wait() 阻塞直到计数归零。
    用法  var wg sync.WaitGroup
          wg.Add(1)              // 启动任务前加计数
          go func() {
              defer wg.Done()    // 任务完成后减计数（defer 确保一定执行）
              // ... 任务代码 ...
          }()
          wg.Wait()              // 阻塞等待所有任务完成

【goroutine（并发）】
───────────────────────────────────────────
  go func() { ... }(参数)
    说明  go 关键字启动新的 goroutine（轻量级线程），立即开始执行。
          匿名函数后跟(参数)表示定义完立即调用。goroutine 跟随主线程异步执行。
    用法  go worker(1, "task")  // 启动异步任务
          // 主线程继续执行，不等待 worker 完成

【切片预分配】
───────────────────────────────────────────
  make([]Type, length)
    说明  预分配指定长度的切片。元素被初始化为零值，长度固定但可通过 append 增长。
          预分配避免后续 append 时的扩容开销，也便于按索引写入。
    用法  results := make([]Message, len(toolCalls))
          // results 长度等于 toolCalls 数量，元素都是零值 Message
          results[0] = Message{...}  // 按索引写入，无并发问题

【append 追加切片】
───────────────────────────────────────────
  slice = append(slice, element)
    说明  向切片追加元素，返回新的切片（可能与原切片共享底层数组）。
          如果切片容量不足，会自动扩容（通常翻倍），这是常见开销来源。
    用法  messages = append(messages, newMsg)
          messages = append(messages, msg1, msg2, msg3)  // 一次追加多个

【defer 延迟执行】
───────────────────────────────────────────
  defer 函数调用
    说明  defer 将函数调用压栈，在包含 defer 的函数返回前按 LIFO 顺序执行。
          常用于资源释放（关闭文件、释放锁）、错误恢复。
    用法  file, _ := os.Open("a.txt")
          defer file.Close()  // 函数结束时自动关闭文件

【context.Context（上下文）】
───────────────────────────────────────────
  context.Context
    说明  上下文接口，携带截止时间、取消信号和请求级数据。
          作为函数第一个参数传递，告知函数如何、何时终止。
    用法  func(ctx context.Context, args string) error
          ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
          defer cancel()  // 用完取消，避免泄露

【fmt.Printf 格式化输出】
───────────────────────────────────────────
  fmt.Printf("格式串", 参数...)
    说明  格式化并打印。格式串包含占位符：%s 字符串、%d 整数、%v 任意值、%% 字面%。
    用法  fmt.Printf("用户: %s, 年龄: %d\n", name, age)

【break 退出循环】
───────────────────────────────────────────
  break
    说明  跳出最近的 for/switch/select 循环或 switch 分支。
    用法  for {
            if condition { break }  // 退出整个循环
          }
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

// AgentEngine 是 Agent 的核心执行引擎
// 它管理：1) LLM Provider（与大模型通信）2) 工具注册表（管理可用工具）3) 工作区目录
type AgentEngine struct {
	provider       provider.LLMProvider // LLM Provider 接口，运行时注入（OpenAI/Claude 等）
	registry       tools.Registry       // 工具注册表，管理所有可用工具
	WorkDir        string               // 工作区根目录，工具操作文件时以此为基准
	EnableThinking bool                 // 是否启用慢思考模式（先推理再行动）
}

// NewAgentEngine 是 AgentEngine 的构造函数
// 构造函数通常以 New 开头，返回类型的指针，负责初始化结构体的所有字段
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// Run 是 Agent 的主循环方法，接收用户提示词并执行多轮对话直到任务完成
// ctx: 上下文，用于传递取消信号和超时控制
// userPrompt: 用户输入的任务描述
func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	// 打印启动日志
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式 (Thinking Phase): %v\n", e.EnableThinking)

	// 初始化对话历史（Context History）
	// 对话历史记录了 Agent 与用户的所有交互，是模型的"记忆"
	// 第一条是系统消息，定义 Agent 的角色和能力；第二条是用户的初始请求
	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem, // RoleSystem = "system"，系统级消息
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser, // RoleUser = "user"，用户消息
			Content: userPrompt,      // 用户传入的任务描述
		},
	}

	turnCount := 0 // 记录 Agent 思考-行动的轮次

	// Agent 主循环：持续执行直到模型决定完成任务（不再调用工具）
	// 这是一个无限循环，必须在内部通过 break 退出
	for {
		turnCount++
		log.Printf("\n========== [Turn %d] 开始 ==========\n", turnCount)

		// 从注册表获取所有可用工具的定义
		// 这些定义会告诉模型有哪些工具可用、每个工具的功能和参数规范
		availableTools := e.registry.GetAvailableTools()

		// ═══════════════════════════════════════════════════════════
		// Phase 1: 慢思考阶段（可选）
		// ═══════════════════════════════════════════════════════════
		// 如果启用慢思考模式，模型会先在"无工具"的环境下进行推理和规划
		// 这样模型可以先想清楚要做什么，再决定调用哪些工具
		if e.EnableThinking {
			log.Println("[Engine][Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")
			// 调用 LLM 生成回复，此时 availableTools = nil（剥夺工具）
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w", err)
			}
			// 如果模型输出了思考内容，打印出来供用户观察
			if thinkResp.Content != "" {
				fmt.Printf("🧠 [内部思考 Trace]: \n%s\n", thinkResp.Content)
				// 将思考结果加入对话历史，供下一阶段参考
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		// ═══════════════════════════════════════════════════════════
		// Phase 2: 行动阶段
		// ═══════════════════════════════════════════════════════════
		// 模型根据上下文和可用工具，决定回复内容和工具调用
		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		// 传入可用工具，让模型知道可以调用哪些工具
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}

		// 将模型的回复加入对话历史
		contextHistory = append(contextHistory, *actionResp)

		// 如果模型有面向用户的回复内容，打印出来
		if actionResp.Content != "" {
			fmt.Printf("🤖 [对外回复]: \n%s\n", actionResp.Content)
		}

		// ═══════════════════════════════════════════════════════════
		// 检查是否需要调用工具
		// ═══════════════════════════════════════════════════════════
		// ToolCalls 是模型请求调用的工具列表
		// 如果为空，说明模型认为任务完成了，不再需要工具
		if len(actionResp.ToolCalls) == 0 {
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			break // 退出主循环，Run 方法正常返回
		}

		log.Printf("[Engine] 模型请求并发调用 %d 个工具...\n", len(actionResp.ToolCalls))

		// ═══════════════════════════════════════════════════════════
		// 并发执行所有工具调用
		// ═══════════════════════════════════════════════════════════
		// observationMsgs 用于收集每个工具的执行结果
		// 预分配切片长度 = 工具调用数量，保证按顺序存放，无需并发锁
		observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))

		// sync.WaitGroup 用于等待所有 goroutine 完成
		var wg sync.WaitGroup

		// 遍历每个工具调用请求，启动独立的 goroutine 并发执行
		for i, toolCall := range actionResp.ToolCalls {
			wg.Add(1) // 每启动一个 goroutine，计数器加 1

			// 启动 goroutine 并发执行工具
			// 匿名函数接收参数 i 和 toolCall，避免循环变量捕获问题
			go func(idx int, call schema.ToolCall) {
				defer wg.Done() // goroutine 结束时通知 WaitGroup，计数器减 1

				log.Printf("  -> [Go-%d] 🛠️ 触发并行执行: %s\n", idx, call.Name)

				// 调用注册表执行工具，并传入上下文（可超时、可取消）
				result := e.registry.Execute(ctx, call)

				// 检查工具执行是否有错误
				if result.IsError {
					log.Printf("  -> [Go-%d] ❌ 工具执行报错: %s\n", idx, result.Output)
				} else {
					log.Printf("  -> [Go-%d] ✅ 工具执行成功 (返回 %d 字节)\n", idx, len(result.Output))
				}

				// 将工具执行结果存入预分配切片对应索引位置
				// 索引 i 对应 toolCall[i] 的结果，保证顺序正确
				observationMsgs[idx] = schema.Message{
					Role:       schema.RoleUser, // 工具结果作为用户消息加入对话
					Content:    result.Output,   // 工具的输出内容
					ToolCallID: call.ID,         // 关联到原来的工具调用
				}
			}(i, toolCall) // 立即调用匿名函数，传入当前参数
		}

		// 等待所有 goroutine 完成
		// wg.Wait() 会阻塞，直到计数器归零
		wg.Wait()
		log.Println("[Engine] 所有并发工具执行完毕，开始聚合观察结果 (Observation)...")

		// 将所有工具执行结果按顺序追加到对话历史
		// 这些结果会在下一轮对话中发送给模型，让它基于观察继续推理
		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}

		// 继续下一轮循环：模型基于工具结果再次决定是回复还是继续调用工具
	}

	// 主循环通过 break 正常退出，Run 方法返回 nil（表示成功）
	return nil
}
