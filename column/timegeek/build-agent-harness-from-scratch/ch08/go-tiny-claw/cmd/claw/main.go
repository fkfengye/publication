/*
Go 语法速查:

── 程序入口 ──
  package main
    说明  声明为可独立运行的可执行程序入口包，Go 要求可执行程序必须使用 main 包
    用法  package main 是程序入口，库包用 package xxx。main 包中必须定义 main() 函数

  func main()
    说明  程序入口函数，Go 程序启动后第一个被调用的函数，无参数无返回值
    用法  func main() { ... }，不能被其他函数调用，Go runtime 自动调用

── 环境变量 ──
  os.Getenv
    说明  读取指定名称的环境变量值，返回字符串，未设置则为空字符串
    用法  key := os.Getenv("ZHIPU_API_KEY")，生产环境常用环境变量存储敏感配置

  os.Getwd
    说明  获取当前工作目录的绝对路径
    用法  wd, err := os.Getwd()，用于解析相对路径为绝对路径

── 并发工具注册 ──
  注册工具到 Agent
    说明  将实现了 BaseTool 接口的工具注册到 Registry，供 Agent 在运行时调用
    用法  registry.Register(tools.NewXxxTool(workDir))，可注册多个工具并行使用

── 上下文 ──
  context.Background
    说明  创建空上下文，通常用作根上下文，贯穿整个请求生命周期
    用法  ctx := context.Background()，在 main 中创建后传递给下游函数
*/

package main

import (
	"context"
	"log"
	"os"

	"github.com/yourname/go-tiny-claw/internal/engine"
	"github.com/yourname/go-tiny-claw/internal/provider"
	"github.com/yourname/go-tiny-claw/internal/tools"
)

// 程序入口函数
func main() {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}

	workDir, _ := os.Getwd()

	// 创建 OpenAI 兼容的智谱 Provider（使用 Zhipu API）
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")
	registry := tools.NewRegistry()

	// 注册文件读写和编辑工具
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir))

	// 开启慢思考，促使大模型一次性规划出并行的工具调用
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)

	prompt := `
	我当前目录下有 a.txt, b.txt, c.txt 三个文件。(如果没有请忽略找不到的报错)
	为了节省时间，请你同时一次性利用工具读取这三个文件，并将它们的内容综合起来告诉我。
	`

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
