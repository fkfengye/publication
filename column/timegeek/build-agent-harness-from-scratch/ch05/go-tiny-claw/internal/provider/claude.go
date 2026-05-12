package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  type struct
    说明  定义结构体，字段小写=包私有
    用法  type ClaudeProvider struct { client anthropic.Client; model string }
          — client 是值类型（非指针），Go SDK 中 Client 通常设计为值类型可安全复制

── 函数与方法 ──
  func NewXxx(...)
    说明  构造函数，本例用 NewZhipuClaudeProvider 命名体现具体实现
    用法  func NewZhipuClaudeProvider(model string) *ClaudeProvider

  panic()
    说明  运行时恐慌，立即终止当前 goroutine 的正常执行，沿调用栈向上传播
    用法  panic("请设置 ZHIPU_API_KEY 环境变量") — 仅用于不可恢复的初始化错误

  (p *Type) Method()
    说明  指针接收者方法
    用法  func (p *ClaudeProvider) Generate(...) (*schema.Message, error)

── 变量与指针 ──
  :=
    说明  短变量声明，自动推断类型
    用法  apiKey := os.Getenv("ZHIPU_API_KEY")

  var
    说明  显式变量声明，可指定类型或延迟赋值
    用法  var anthropicMsgs []anthropic.MessageParam — 声明空切片，值为 nil

  &Struct{}
    说明  创建结构体并取地址
    用法  &ClaudeProvider{client: anthropic.NewClient(...), model: model}

  _ = expr
    说明  显式忽略返回值，表示有意不处理
    用法  _ = json.Unmarshal(tc.Arguments, &inputMap) — Unmarshal 可能失败但忽略错误

── 复合类型 ──
  []Type
    说明  切片
    用法  []anthropic.MessageParam, []schema.Message, []anthropic.ToolUnionParam

  map[string]interface{}
    说明  string 到任意类型的映射，常用于解析动态 JSON
    用法  var inputMap map[string]interface{} — 零值为 nil，json.Unmarshal 会分配内存

  []Type{...}
    说明  切片字面量，初始化时赋值
    用法  []anthropic.TextBlockParam{{Text: systemPrompt}}

── 控制流 ──
  switch/case
    说明  多路分支，Go 中每个 case 默认自带 break，不需要显式写
    用法  switch msg.Role { case schema.RoleSystem: ... case schema.RoleUser: ... }

  for range {}
    说明  遍历切片
    用法  for _, msg := range msgs { ... } — 遍历消息列表做格式转换

  for _, v := range
    说明  遍历切片，_ 丢弃索引
    用法  for _, block := range resp.Content { ... } — 遍历 API 响应内容块

  if/else
    说明  条件分支
    用法  if msg.ToolCallID != "" { ... } else { ... } — 区分纯文本和工具结果

  if _, ok := m[k]; ok
    说明  map 安全读取惯用写法，ok 为 true 表示 key 存在
    用法  if properties, ok := m["properties"].(map[string]interface{}); ok { ... }

── 变量与指针 ──
  *Type 指针
    说明  指针类型，用于可选字段
    用法  anthropic.String(toolDef.Description) — SDK 用 *string 表示可选字段，辅助函数创建指针

  &Struct{...}
    说明  取结构体地址
    用法  anthropic.ToolUnionParam{OfTool: &tp} — 取 tp 地址传给联合类型字段

── 格式化 ──
  json.Marshal / json.Unmarshal
    说明  JSON 序列化/反序列化
    用法  json.Marshal(block.Input) — 将 map 序列化为 JSON 字节；json.Unmarshal 反之

  fmt.Errorf + %w
    说明  错误包装，%w 保留原始错误链
    用法  fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 访问其他包导出的标识符
    用法  anthropic.Client, anthropic.MessageNewParams, option.WithAPIKey, schema.Message
*/

// 定义 ClaudeProvider 结构体，封装 Anthropic SDK 客户端和模型名
type ClaudeProvider struct {
	client anthropic.Client
	model  string
}

// 创建智谱 Anthropic 兼容接口的 Provider，通过环境变量获取 API Key
func NewZhipuClaudeProvider(model string) *ClaudeProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}
	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return &ClaudeProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

// Generate 将内部消息格式转为 Anthropic API 格式发送请求，再将响应转回内部格式
func (p *ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var anthropicMsgs []anthropic.MessageParam
	var systemPrompt string

	// 将内部消息逐条转换为 Anthropic SDK 的消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			// Anthropic API 的 system prompt 是顶层字段，不放在 messages 中
			systemPrompt = msg.Content
		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 带 ToolCallID 的消息 = 工具执行结果，用 ToolResultBlock 表示
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			} else {
				// 普通用户消息
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case schema.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			// 将 ToolCall 转为 Anthropic 的 ToolUseBlock
			for _, tc := range msg.ToolCalls {
				var inputMap map[string]interface{}
				_ = json.Unmarshal(tc.Arguments, &inputMap)
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: inputMap,
					},
				})
			}
			if len(blocks) > 0 {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(blocks...))
			}
		}
	}

	// 将内部工具定义转为 Anthropic SDK 的 Tool 格式
	var anthropicTools []anthropic.ToolUnionParam
	for _, toolDef := range availableTools {
		var properties map[string]any
		var required []string

		// 从 InputSchema 中提取 properties 和 required 字段
		if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
			if p, ok := m["properties"].(map[string]interface{}); ok {
				properties = p
			}
			if r, ok := m["required"].([]string); ok {
				required = r
			}
		}

		tp := anthropic.ToolParam{
			Name:        toolDef.Name,
			Description: anthropic.String(toolDef.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: properties,
				Required:   required,
			},
		}
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &tp})
	}

	// 构造 API 请求参数
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMsgs,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	// 调用 Anthropic API
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)
	}

	// 将 Anthropic 响应转为内部消息格式
	resultMsg := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			resultMsg.Content += block.Text
		case "tool_use":
			// 将 tool_use block 的 Input map 序列化为 JSON 字节
			argsBytes, _ := json.Marshal(block.Input)
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: argsBytes,
			})
		}
	}

	return resultMsg, nil
}
