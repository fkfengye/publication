/*
Go 语法速查:

── 包与导入 ──
  package main
    说明  每个可执行程序必须有一个 main 包，编译器由此生成可执行文件
    用法  package main 声明本文件属于 main 包；一个目录下所有 .go 文件必须属于同一个包

  import (...)
    说明  分组导入多个包，每个包名用双引号包裹
    用法  import ("os"; "log") 一次导入多个包。Go 要求导入的包必须被使用，
          否则编译报错。分组导入是 Go 的标准风格

── 变量与初始化 ──
  os.Getenv("KEY")
    说明  读取环境变量值，若不存在返回空字符串
    用法  apiKey := os.Getenv("ZHIPU_API_KEY") — 空值配合 if 检查可判断环境变量是否设置

  os.Getwd()
    说明  获取当前工作目录路径，返回 (string, error)
    用法  workDir, _ := os.Getwd() — 用 _ 忽略错误，适合仅在理论上可能失败的调用

  :=
    说明  短变量声明，在函数内部推断变量类型并赋值
    用法  eng := engine.NewAgentEngine(...) — 省去 var 关键字和类型声明，Go 中最常用的变量声明方式

  零值初始化
    说明  声明结构体时未显式赋值的字段自动为零值
    用法  &tools.Registry{} 中 tools 字段默认为 nil，但 NewRegistry() 内部会初始化 map

── 流程控制 ──
  for range
    说明  遍历数组、切片、map、通道
    用法  本文件未使用，但 ch07 引擎代码大量使用 for range 遍历消息历史和工具列表

── 函数与方法 ──
  log.Fatal
    说明  打印日志后调用 os.Exit(1) 立即终止程序
    用法  log.Fatal("请先导出 ZHIPU_API_KEY 环境变量") — 在配置检查失败时使用，
          确保程序在无效状态下不会继续运行

── 类型系统 ──
  方法值作为参数
    说明  方法（有接收者的函数）可以作为值传递，与其他函数值兼容
    用法  tools.NewReadFileTool(workDir) 返回实现了 BaseTool 接口的实例，
          通过 registry.Register() 注册到全局工具列表

── 空行字符串字面量 ──
  反引号 `...`
    说明  原始字符串，内容不转义，支持跨行
    用法  多行 prompt 使用反引号包裹，避免 \n 转义

── 结构体指针 ──
  &Type{}
    说明  创建结构体实例并获取其指针
    用法  &tools.Registry{} 调用 NewRegistry 获取的是指针类型，便于方法修改接收者状态
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

// 程序入口：初始化 Provider、注册工具、启动 Agent 引擎执行文件编辑任务
func main() {

	// 校验环境变量：智谱 API Key 必须设置
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}

	// 获取当前工作目录，作为所有文件操作的基准路径
	workDir, _ := os.Getwd()

	// 初始化 OpenAI 兼容接口的 Provider（底层对接智谱 GLM 模型）
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")
	// 创建工具注册表
	registry := tools.NewRegistry()

	// 注册四个工具：读文件、写文件、执行 Shell 命令、编辑文件
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir)) // 精确字符串替换工具，ch07 新增

	// 创建 Agent 引擎，开启慢思考模式（EnableThinking=false 表示跳过内部思考阶段）
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 构造任务 prompt：要求模型读取 server.go，定位 TODO 注释并替换 if 语句
	prompt := `
	我当前目录下有一个 server.go 文件。
	请帮我把里面 "TODO: 增加鉴权逻辑" 下面的那个 if 语句，整个替换为：
	if user == nil {
		fmt.Println("Forbidden!")
		return
	}
	`

	// 执行 Agent 任务
	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
