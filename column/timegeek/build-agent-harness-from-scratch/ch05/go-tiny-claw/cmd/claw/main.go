package main

import (
	"context"
	"log"
	"os"

	"github.com/yourname/go-tiny-claw/internal/engine"
	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

/*
Go 语法速查:

── 包与导入 ──
  package main
    说明  每个可执行程序必须有一个 main 包，编译器由此生成可执行文件
    用法  package main 声明本文件属于 main 包；一个目录下所有 .go 文件必须属于同一个包

  func main()
    说明  程序入口点，Go 运行时自动调用，无参数无返回值
    用法  每个 main 包必须有且只有一个 func main()，os.Args 获取命令行参数

── 变量与指针 ──
  :=
    说明  短变量声明，声明 + 赋值一步完成，编译器自动推断类型
    用法  workDir, _ := os.Getwd() — 多返回值用 := 接收，仅函数内可用

  _
    说明  空白标识符，丢弃不需要的值，避免编译报"变量未使用"
    用法  _, _ := os.Getwd() — 第二个返回值是不关心的 error，用 _ 丢弃

  &Struct{...}
    说明  创建结构体并取地址，一步返回指针
    用法  p := &mockProvider{} — & 紧贴类型，{} 内按字段名赋值

── 上下文 ──
  context.Background()
    说明  创建空的根 context，通常用于程序入口、测试、初始化阶段
    用法  作为第一个参数传给 Run：eng.Run(context.Background(), "任务描述")

── 环境变量 ──
  os.Getenv(key)
    说明  读取环境变量，不存在时返回空字符串
    用法  os.Getenv("ZHIPU_API_KEY") — key 区分大小写（Windows 不区分，Linux 区分）

── 错误处理 ──
  log.Fatal / log.Fatalf
    说明  输出日志后调用 os.Exit(1) 立即终止程序，defer 语句不会执行
    用法  log.Fatal("msg") 用于不可恢复的初始化错误；log.Fatalf 支持格式化
*/

// 程序入口
func main() {
	// 校验必要环境变量是否已设置
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}

	// 获取当前工作目录作为工作区路径
	workDir, _ := os.Getwd()

	// 创建智谱 OpenAI 兼容的 LLM Provider，指定模型名
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	// 创建工具注册表并注册文件读取工具
	registry := tools.NewRegistry()

	readFileTool := tools.NewReadFileTool(workDir)
	registry.Register(readFileTool)

	// 初始化 Agent 引擎，传入 provider、工具注册表、工作区和 thinking 开关
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	prompt := "请调用工具读取一下当前工作区目录下 hello.txt 文件的内容，并用一句话向我总结它说了什么。"

	// 执行 Agent 任务
	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
