# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 仓库性质

这是**个人出版物的配套源码仓库**，不是单一应用项目。三个顶层目录对应三种出版物形态：

- `book/` — 图书配套代码（目前空，预留）
- `column/` — 长专栏（imooc、timegeek 等平台）
- `micro-column/` — 短专栏/小专题

每个子目录是一个独立专栏，专栏下按 `ch01/ch02/...` 切分。**章节之间没有运行时依赖**——每个章节都是独立的 Go module 或独立 demo，互不影响。

## Go 项目组织约定

- 教学专栏用相同的 module 名 `github.com/yourname/go-tiny-claw`，方便章节之间对比演进
- 标准布局：`cmd/<name>/main.go` + `internal/<pkg>/`
- 外部依赖按章节渐进：ch01–ch07 零依赖（mock provider）；ch08+ 引入 `anthropic-sdk-go` 和 `openai-go/v3`；更后面章节引入 SDK 工具、MCP、状态文件等
- Go 版本：**1.25**
- 注释规范：调用 `.claude/skills/go-code-annotation/SKILL.md`，按"文件顶部语法速查 + 代码块功能描述 + 关键逻辑说明"三层体系

## 高频命令

没有顶层 Makefile。**每个模块独立运行**：

```bash
# 进入某章节
cd column/timegeek/build-agent-harness-from-scratch/ch08/go-tiny-claw

go mod tidy       # 首次拉依赖
go run ./cmd/claw # 跑 main 包
go test ./...     # 跑该模块全部测试
go vet ./...      # 静态检查
gofmt -w .        # 格式化
```

测试主要分布在 `column/imooc/go-50tips/sources/`（go-testing 系列）、`column/timegeek/go-advanced-course/ch25/`（测试方法论专章）、`micro-column/go-testing-journey/`。

## Big-picture：build-agent-harness-from-scratch 系列

这是仓里最系统、最有演进价值的系列——**22 章**讲"用 Go 从零构建 AI Agent Harness"。专栏名 `go-tiny-claw`（致敬 Claude Code 内部代号 tiny-claw）。

**演进脉络：**

| 阶段 | 章节 | 关键能力 |
|------|------|----------|
| 最小引擎 | ch01–ch07 | ReAct 主循环、mock provider/registry、并发工具执行（`sync.WaitGroup` + 索引预分配切片） |
| 接入 LLM | ch08 | Provider 抽象（Claude + OpenAI 双协议）、4 个真实工具（read/write/bash/edit）、慢思考 + 行动双阶段 |
| 实战化 | ch09–ch19 | 多 Provider 适配、状态外部化（PLAN.md / TODO.md）、Reporter（飞书）、工具扩容 |
| 高级能力 | ch20–ch22 | 流式输出、记忆持久化、子 Agent 等 |

**六边形架构**（ch08 起稳定成形）：

```
        engine (ReAct 主循环：思考 → 行动 → 并发执行 → 回灌)
        /                            \
provider (LLM 防腐层)        tools (registry + BaseTool)
        \                            /
            schema (中立 Message / ToolCall / ToolDefinition)
```

**核心设计哲学**（来自 ch09+ README）：Harness over Framework——壁垒不在 LLM API 调用，而在工具调度、上下文管理、安全拦截。状态外部化（持久化到 PLAN.md / TODO.md），拒绝内存状态机。

阅读后续章节时，重点对照"loop 多了什么分支"、"schema 多了什么字段"、"tools 多了什么 BaseTool"。

## 协作规范继承

以下规则从用户级 `~/.claude/CLAUDE.md` 继承，在本仓继续生效：

- 用户称呼为「帅帅」，所有内容用中文
- **不自动执行 git 操作**（add/commit/push 等必须用户显式授权）
- 需求不明确时主动澄清；被质疑时从需求出发验证
- 代码 Review/分析/设计时追溯完整调用链、考虑上下游
- 优先组合而非继承；禁止过度设计

## 项目级约定

见 `.claude/CLAUDE.md`：

- Go 代码注释统一调用 `go-code-annotation` skill
- 顶层目录（专栏→章节）的拆解逻辑
