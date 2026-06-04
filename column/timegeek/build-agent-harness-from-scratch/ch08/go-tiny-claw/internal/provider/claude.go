/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type ClaudeProvider struct { client anthropic.Client; model string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针，负责初始化结构体
    用法  func NewZhipuClaudeProvider(model string) *ClaudeProvider { return &ClaudeProvider{...} }

── 环境变量 ──
  os.Getenv
    说明  读取环境变量，未设置返回空字符串
    用法  apiKey := os.Getenv("ZHIPU_API_KEY")，敏感信息通过环境变量注入

── 类型断言 ──
  if v, ok := m[key].(Type); ok { ... }
    说明  从接口类型安全提取具体类型，ok 为 false 表示类型不匹配
    用法  m["properties"].(map[string]interface{})，用于解析动态 JSON 结构

── 切片追加 ──
  append(slice, element)
    说明  向切片追加元素，返回新切片
    用法  anthropicMsgs = append(anthropicMsgs, ...)

── JSON 处理 ──
  json.Marshal / json.Unmarshal
    说明  JSON 序列化/反序列化
    用法  argsBytes, _ := json.Marshal(block.Input) 将 map 转回 JSON 字节

── switch 多返回值 ──
  switch v := expr.(type) { ... }
    说明  类型分支，expr 必须是接口类型，根据实际类型匹配分支
    用法  switch block.Type { case "text": ...; case "tool_use": ... }

── 错误处理 ──
  error
    说明  内置接口 type error interface { Error() string }
    用法  return nil, fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)
*/

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

// ClaudeProvider 使用 Anthropic SDK 与 Claude 模型交互（兼容智谱 Zhipu API）
type ClaudeProvider struct {
	client anthropic.Client
	model  string
}

// NewZhipuClaudeProvider 构造函数，创建智谱 API 的 Claude 兼容 Provider
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

// NewMiniMaxClaudeProvider 构造函数，创建智谱 API 的 Claude 兼容 Provider
func NewMiniMaxClaudeProvider(model string) *ClaudeProvider {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		panic("请设置 MINIMAX_API_KEY 环境变量")
	}
	baseURL := "https://api.minimaxi.com/anthropic"
	return &ClaudeProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

// Generate 是 Provider 层的"翻译官"：把内部中立消息格式翻成 Anthropic 协议，
// 调用 API 后再把响应翻回内部 schema.Message。本方法分四步：
//  1. msgs 转换    : schema.Message → anthropic.MessageParam（含 system 单独提取）
//  2. 工具定义转换: schema.ToolDefinition → anthropic.ToolParam
//  3. HTTP 调用   : p.client.Messages.New(ctx, params)
//  4. 响应解析    : resp.Content blocks → schema.Message（text 拼到 Content，tool_use 拼到 ToolCalls）
func (p *ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	// 1) 转换消息历史：内部 schema 格式 → Anthropic 协议格式
	var anthropicMsgs []anthropic.MessageParam
	var systemPrompt string // Anthropic 把 system 单独放在 params.System 字段，不进 messages 数组

	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			// system 消息不进循环数组，只缓存起来最后设到 params.System
			systemPrompt = msg.Content
		case schema.RoleUser:
			// 区分"普通 user 文本"和"工具结果回调"——通过 ToolCallID 是否非空判断
			if msg.ToolCallID != "" {
				// 工具结果：必须是 user role + ToolResultBlock，并通过 tool_use_id 关联到上一轮的 tool_use
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			} else {
				// 普通用户文本
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case schema.RoleAssistant:
			// assistant 消息可能同时含"文本回复"和"多个工具调用请求"——统一成 blocks 数组
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			// 把历史 tool_calls 翻译成 tool_use blocks
			for _, tc := range msg.ToolCalls {
				var inputMap map[string]interface{}
				// Arguments 是 json.RawMessage，Anthropic SDK 要 map——做一次 JSON 往返
				_ = json.Unmarshal(tc.Arguments, &inputMap)
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: inputMap,
					},
				})
			}
			// 没有任何内容（空文本+无工具调用）就不进消息列表，避免空消息触发 SDK 报错
			if len(blocks) > 0 {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(blocks...))
			}
		}
	}

	// 2) 转换工具定义：内部 schema.ToolDefinition → Anthropic 协议 ToolParam
	var anthropicTools []anthropic.ToolUnionParam
	for _, toolDef := range availableTools {
		var properties map[string]any
		var required []string

		// InputSchema 在 schema 包里是 interface{}，实际上每个工具填的是 map[string]interface{}
		// 用类型断言拆出 properties 和 required 两部分，分别填入 Anthropic 的对应字段
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

	// 3) 构造请求参数
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096, // 智谱 GLM 也走这个字段，4096 是稳妥值
		Messages:  anthropicMsgs,
	}

	// 把缓存好的 system prompt 单独设到 params.System（Anthropic 协议规定）
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	// 挂载工具；空切片不发送，避免 SDK 触发"无工具但声明了 tools"的校验
	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	// 4) 真正发 HTTP 请求到智谱 Zhipu（BaseURL 已在 NewZhipuClaudeProvider 中切好）
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)
	}

	// 5) 解析响应：Anthropic 返回的 resp.Content 是 block 数组，
	//    文本 block → 拼到 Content；tool_use block → 构造 ToolCall 塞进 ToolCalls
	resultMsg := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			// 多段文本块按顺序拼接（Anthropic 极少返回多段，但要做容错）
			resultMsg.Content += block.Text
		case "tool_use":
			// 工具调用：把 SDK 给的 Input (map) 再 marshal 回 json.RawMessage
			// 这样下游 tools 包拿到的 Arguments 仍是原始 JSON 字节，可以延迟解析
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
