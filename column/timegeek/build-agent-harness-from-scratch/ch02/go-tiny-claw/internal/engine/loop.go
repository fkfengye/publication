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
    说明  定义结构体，字段名首字母大写=导出，小写=私有
    用法  type AgentEngine struct { provider provider.LLMProvider } — 字段 provider 是小写私有的，
          包外不可直接访问，通过构造函数初始化

  &Struct{...}
    说明  & 后跟结构体字面量，一步完成"创建结构体 + 取地址"，返回指针
    用法  return &AgentEngine{provider: p, registry: r, WorkDir: workDir} — 常用在构造函数中，
          比先 new 再赋值更简洁，{} 内按字段名赋值顺序不限

── 函数与方法 ──
  (e *Type) Method()
    说明  给类型绑定方法，e 称为接收者。*Type 是指针接收者，可修改原始对象
    用法  func (e *AgentEngine) Run(...) error — 调用时 e.Run()，Go 自动取地址/解引用。
          需要修改接收者状态时用指针接收者，只读且小对象可用值接收者

  func NewXxx(...)
    说明  Go 无构造函数语法，约定用 New + 类型名 命名的工厂函数返回初始化好的实例
    用法  func NewAgentEngine(p ..., r ..., workDir string) *AgentEngine。返回指针给外部使用，
          内部私有字段在函数内赋值

  多返回值
    说明  Go 原生支持多返回值，最常用的模式是返回 (result, error)
    用法  func Run(...) error — 这里只返回 error，成功返回 nil；若需要返回值，
          写成 func Run(...) (Result, error)，调用方用 r, err := Run(...) 接收

── 变量与指针 ──
  :=
    说明  短变量声明，自动推断类型并赋值，仅函数内可用
    用法  turnCount := 0 等价于 var turnCount int = 0。简洁但要注意作用域：
          if 内用 := 创建的变量在 if 外不可见

  *ptr
    说明  解引用指针，取出指针指向的值
    用法  *responseMsg — responseMsg 是 *schema.Message 指针，*responseMsg 取出 Message 值。
          append 需要值类型，所以需解引用后再追加到切片

── 复合类型 ──
  []Type{...}
    说明  切片字面量，在声明时直接初始化切片的元素
    用法  contextHistory := []schema.Message{{Role: schema.RoleSystem, Content: "..."}}
          外层 {} 是切片，内层 {} 是 Message 结构体。未指定长度，长度由初始元素个数决定

── 控制流 ──
  for {}
    说明  无条件的无限循环，等价于其他语言的 while(true)
    用法  for { ... } — 必须用 break 或 return 退出，否则程序永远卡住。
          常用于服务器主循环、事件轮询

  for range {}
    说明  遍历切片/map/字符串/通道，每次迭代返回索引和元素
    用法  for _, toolCall := range responseMsg.ToolCalls — 用 _ 丢弃索引，只取元素值。
          遍历的是副本拷贝，修改 toolCall 不影响原切片

  break
    说明  跳出当前最内层循环，程序继续执行循环后的代码
    用法  if len(responseMsg.ToolCalls) == 0 { break } — 没有工具调用时跳出 for{} 循环，
          结束 Agent 主循环

  if err != nil
    说明  Go 错误处理的标准惯用写法，没有异常机制，错误通过返回值传递
    用法  responseMsg, err := e.provider.Generate(...); if err != nil { return err }
          Go 要求你必须检查 error，不能静默忽略

── 错误处理 ──
  fmt.Errorf + %w
    说明  创建带格式的错误消息，%w 包装原始错误，保留错误链供上层检查
    用法  return fmt.Errorf("模型生成失败: %w", err) — 调用方用 errors.Is / errors.As
          可以追溯到原始错误类型，用于分类处理

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 引用其他包的导出类型
    用法  provider.LLMProvider, tools.Registry — provider 和 tools 是 import 的包名。
          只有首字母大写的类型/函数才能在包外使用

── 内置函数 ──
  len()
    说明  返回集合的长度：切片元素数、map 键数、字符串字节数
    用法  len(responseMsg.ToolCalls) — 判断切片是否有元素。空切片/nil 切片 len 都返回 0

  append()
    说明  向切片末尾追加元素，可能触发扩容分配新底层数组
    用法  contextHistory = append(contextHistory, *responseMsg) — 必须用变量接收返回值，
          append 不修改原切片，返回新切片（可能指向新内存）

── 格式化 ──
  log.Printf / log.Fatalf
    说明  格式化日志输出。Printf 输出后继续执行，Fatalf 输出后调用 os.Exit(1) 终止进程
    用法  log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir) — %s 替换为字符串值。
          Fatalf 只应在不可恢复的致命错误时使用

  %占位符
    说明  fmt 包的格式化动词，控制变量如何转为字符串
    用法  %s 字符串，%d 十进制整数，%v 通用格式（自动选类型默认显示），%+v 带字段名的结构体
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
