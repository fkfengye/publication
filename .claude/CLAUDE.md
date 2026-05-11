# 项目说明

## Go 代码注释规范

本项目使用 `go-code-annotation` 技能为 Go 代码添加注释。详见 `.claude/skills/go-code-annotation.md`。

## 目录结构

- `column/timegeek/build-agent-harness-from-scratch/` — 从零构建 Agent 框架系列文章代码
  - `ch02/go-tiny-claw/` — 第二章示例：最小 Agent 引擎
    - `cmd/claw/main.go` — 程序入口
    - `internal/engine/loop.go` — Agent 主循环引擎
    - `internal/provider/interface.go` — LLM Provider 接口
    - `internal/tools/registry.go` — 工具注册表接口
    - `internal/schema/message.go` — 核心数据结构