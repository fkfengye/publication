# go-tiny-claw: ch08 极简智能体驾驭引擎 — 项目分析

> 对应源码：本目录下 `cmd/`、`internal/`、`a.txt / b.txt / c.txt`
>
> 第 8 章示例：最小可运行的 Coding Agent 引擎（致敬 Claude Code 内部代号 tiny-claw）。
>
> 分析时间：2026/06/03

---

## 一、项目定位

第 8 章的实战代码，主题是**「Harness over Framework」**——把"用户给一段自然语言任务 → 模型思考 → 模型决定调哪些工具 → 工具在本地工作区执行 → 结果回灌给模型"这条链路，用 **600 行 Go** 跑通。

依赖：

- `github.com/anthropics/anthropic-sdk-go v1.30.0`
- `github.com/openai/openai-go/v3 v3.30.0`
- Go 1.25

代码全部带大段中文注释（文件顶部有 `Go 语法速查` 文档块），是典型的"教学型"工程：边看代码边学 Go 语法。

---

## 二、目录与分层

```
ch08/go-tiny-claw/
├── cmd/claw/main.go              # 程序入口：装配组件 + 启动 Agent
├── internal/
│   ├── engine/loop.go            # Agent 主循环（核心）
│   ├── provider/
│   │   ├── interface.go          # LLMProvider 抽象接口
│   │   ├── claude.go             # Anthropic 协议实现（智谱代理）
│   │   └── openai.go             # OpenAI 协议实现（智谱代理）
│   ├── schema/message.go         # 中立数据结构（消息/工具调用/工具定义）
│   └── tools/
│       ├── registry.go           # 工具注册中心
│       ├── bash.go               # bash 命令执行
│       ├── read_file.go          # 读文件
│       ├── write_file.go         # 写文件
│       └── edit_file.go          # 局部编辑（含四层模糊匹配）
├── a.txt / b.txt / c.txt         # 演示用的三个小文件
└── go.mod / go.sum
```

**六边形架构**：

- `engine` 是中心
- `provider` 和 `tools` 是两侧适配器
- `schema` 是两边共用的"语言"（中立数据结构）
- `main` 是装配根

---

## 三、核心数据模型 (`schema/message.go`)

整个系统只用了 **4 个类型**，把"模型对话 + 工具调用"抽象得极简：

| 类型 | 作用 |
|------|------|
| `Role` | 三种角色常量：`system` / `user` / `assistant` |
| `Message` | 一条对话消息：角色 + 文本 + 可选 `ToolCalls` + 可选 `ToolCallID` |
| `ToolCall` | 模型请求的工具调用：ID + 工具名 + `json.RawMessage` 参数 |
| `ToolDefinition` | 工具元信息：名称 + 描述 + JSON Schema |
| `ToolResult` | 工具执行结果：ID + 输出文本 + `IsError` 标志 |

**关键设计**：参数用 `json.RawMessage` 透传，不做反序列化。`provider` 层不关心每个工具的参数形状，只需原样塞进厂商 SDK；只有具体工具自己解析。

---

## 四、Provider 层（模型适配器）

### 4.1 接口

```go
// internal/provider/interface.go
type LLMProvider interface {
    Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
```

**所有厂商差异都收敛在 `Generate` 内部**。

### 4.2 两个实现

`ClaudeProvider` 和 `OpenAIProvider` 走的是**智谱 Zhipu**（`https://open.bigmodel.cn/api/paas/v4/`），通过 `WithBaseURL` 把流量切到智谱，`ZHIPU_API_KEY` 是统一环境变量。

两者都做三件事：

1. **内部 `Message` → 厂商格式** 的转换（system 字段、user text、user tool_result、assistant text + tool_use/tool_calls）
2. **内部 `ToolDefinition` → 厂商 tool schema** 的转换
3. **调用 SDK → 解析响应 → 还原为内部 `Message`**

这是**Anti-Corruption Layer（防腐层）** 模式：把外部 SDK 的脏数据挡在边界外，引擎层只跟中立 schema 打交道。

---

## 五、Tools 层（工具生态）

### 5.1 抽象

```go
// internal/tools/registry.go
type BaseTool interface {
    Name() string
    Definition() schema.ToolDefinition
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry interface {
    Register(BaseTool)
    GetAvailableTools() []schema.ToolDefinition
    Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}
```

实现 `registryImpl` 用 `map[string]BaseTool` 做名字索引，**查找 O(1)**。重名注册会 warn（用 `log.Printf`，符合项目允许的 info/warn 习惯）。

### 5.2 四个具体工具

| 工具 | 能力 | 关键实现 |
|------|------|---------|
| `read_file` | 读相对路径文件 | 8000 字节截断 |
| `write_file` | 创建 / 覆盖写文件 | `MkdirAll` 自动建父目录，权限 `0644` |
| `edit_file` | 局部字符串替换 | **四层模糊匹配** + 唯一性校验 |
| `bash` | 执行 shell 命令 | `exec.CommandContext` + 30 秒超时 + 8000 字节截断 |

### 5.3 `edit_file` 的四层模糊匹配（亮点）

`fuzzyReplace` 实现了**四层降级匹配**：

| 优先级 | 策略 | 适用场景 |
|--------|------|---------|
| L1 | 精确匹配 | 模型给的 `old_text` 跟文件完全一致 |
| L2 | `\r\n → \n` 归一化 | Windows 换行导致不匹配 |
| L3 | `TrimSpace` | 行尾空白差异 |
| L4 | 逐行去缩进滑动窗口 | 缩进 / 空格不一致 |

**唯一性约束**：任何一层匹配出 0 处或 >1 处都报错，要求模型补充上下文——避免"误改大片代码"的灾难。

---

## 六、Engine 层（核心大脑）

`engine/loop.go` 的 `Run` 方法是整个 Agent 的**主循环**。完整调用链：

```
main.Run
  └─ engine.Run
        ├─ (Turn N 开始)
        │   ├─ Phase 1 [可选] 慢思考：Generate(messages, tools=nil)
        │   │   └─ 拿回 thinkResp → 打印 🧠 → 追加到 history
        │   │
        │   ├─ Phase 2 行动：Generate(messages, availableTools)
        │   │   └─ 拿回 actionResp → 打印 🤖 → 追加到 history
        │   │
        │   ├─ 决策：len(ToolCalls) == 0 ? break : 继续
        │   │
        │   ├─ 并发执行工具：
        │   │   for i, call := range ToolCalls {
        │   │       go func(idx, call) {
        │   │           defer wg.Done()
        │   │           observationMsgs[idx] = registry.Execute(ctx, call)
        │   │       }(i, call)
        │   │   }
        │   │   wg.Wait()  // 等所有 goroutine
        │   │
        │   └─ 观察追加：history += observationMsgs
        │
        └─ (Turn N+1)
```

### 6.1 关键设计点

**1. 双阶段推理（Thinking + Action）**

- Phase 1 故意传 `tools=nil`，让模型"先想清楚再动手"
- 想完再 Phase 2 给工具——单次回复里，模型更容易**一次性规划多个并行工具调用**
- 这就是 `main.go` 里"请同时一次性利用工具读取这三个文件"能成功的根因

**2. 并发工具执行**

- `sync.WaitGroup` + `make([]Message, len(toolCalls))` 预分配切片
- **按索引写**而不是 append，规避了并发 slice 写竞争
- 每个工具都 `defer wg.Done()`，确保 panic 也能正确递减计数

**3. History 维护规则**

- assistant 的 tool_use（tool_calls）原样进 history
- tool 的结果用 `RoleUser` + `ToolCallID` 进 history
- 这样下一轮 LLM 能看到"上次我请求了什么 → 工具回了什么"

**4. 退出条件**

- 只有 `len(actionResp.ToolCalls) == 0` 才 break
- **没有最大轮次限制、没有 token 限制**——生产化时需要补

---

## 七、main 入口的装配

```go
func main() {
    1. 校验 ZHIPU_API_KEY
    2. 拿 workDir = os.Getwd()
    3. NewZhipuOpenAIProvider("glm-4.5-air")     // 智谱 GLM
    4. NewRegistry() + Register × 4 (read/write/bash/edit)
    5. NewAgentEngine(provider, registry, workDir, enableThinking=true)
    6. eng.Run(ctx, prompt)  // 提示：让 Agent 并行读 a/b/c.txt
}
```

`enableThinking=true` 是默认值——印证了第 8 章的核心教学点：**慢思考 + 并发工具调用是这套 Agent 的两大特征**。

---

## 八、整体做了一件什么事

用一句话概括：

> 把"用户给一段自然语言任务 → 反复『模型思考 → 模型决定调哪些工具 → 工具在本地工作区执行 → 结果回灌给模型』循环 → 直到模型说『做完了』"这条链路，用 600 行 Go 跑通。

跟 ch02 的最小引擎相比，ch08 的能力增量是：

| 能力 | ch02 | ch08 |
|------|------|------|
| 多厂商 LLM 适配 | 单一 | 抽象 `LLMProvider` 接口 + 双实现 |
| 工具生态 | 无 | 注册中心 + 4 个真实可用工具 |
| 工具调用 | 无 | 模型 → JSON → 工具 → 结果 → 回灌 |
| 并发 | 无 | 多工具 `goroutine` 并行执行 |
| 思考模式 | 无 | Phase 1 无工具推理 + Phase 2 行动 |
| 文件编辑鲁棒性 | 无 | 四层模糊匹配 + 唯一性校验 |
| 输出控制 | 无 | 8000 字节截断 + 超时保护 |

---

## 九、可挑刺的地方（实事求是）

按"尊重事实、纠正错误"的协作原则，标注几个**真实存在的小瑕疵**：

### 9.1 `NewZhipuOpenAIProvider` 没校验 `apiKey == ""`

- `main.go` 提前做了 `log.Fatal`，所以暂时不 panic
- 但 `ClaudeProvider` 的构造函数里有 `panic`，**两个不一致**
- 项目原则"全链路只在一处进行防御式编程"——入口校验应该只在一处
- 此外 `log.Fatal` 风格与项目其它 `AdpBusinessException` 规范不一致（教学项目，规则不严可理解）

### 9.2 `engine/loop.go` 的慢思考没有 token 预算

- 如果模型陷入"反复思考不行动"会无限循环
- 生产化需要加 `MaxTurns` 或 `MaxTokens`

### 9.3 `BashTool` 30 秒超时被硬编码

- 应该作为参数注入或配置化

### 9.4 `Registry.Execute` 用 `Output string + IsError bool` 表达错误

- `IsError` 永远没在 `engine.loop.go` 里有差异化处理
- 严格来说可以升级为 `ToolResult` 自带多模态 / 结构化字段

### 9.5 `context` 透传了但没设置 `WithCancel`

- `main.go` 用的是 `context.Background()`，程序只能等 `Run` 完或 panic 才能退出
- 缺一个 `SIGINT → cancel()` 的信号处理

---

## 十、读后感

这个项目是一个**结构清晰、注释丰富、教学导向**的 Agent 最小引擎。它示范了 4 件工程上非常重要的事：

1. **中立 schema 解耦** provider 和 tools
2. **接口 + 依赖注入** 让 LLM 厂商可替换
3. **主循环 + 并发工具** 表达 Agent 范式
4. **鲁棒性细节**（截断、超时、模糊匹配）藏在工具实现里

下一章（ch09 之后）通常会引入：流式输出、消息压缩、子 Agent、记忆持久化、MCP 协议等。**这一章是"骨架"——后面的章节都是在骨架上长肉**。建议读后续章节时，重点对照"loop 多了什么分支"、"schema 多了什么字段"。
