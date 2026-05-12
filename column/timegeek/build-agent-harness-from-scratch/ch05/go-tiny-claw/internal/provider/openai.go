package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  type struct
    说明  定义结构体，client 字段是值类型（非指针）
    用法  type OpenAIProvider struct { client openai.Client; model string }

── 函数与方法 ──
  func NewXxx(...)
    说明  构造函数
    用法  func NewZhipuOpenAIProvider(model string) *OpenAIProvider

  panic()
    说明  运行时恐慌，立即终止当前 goroutine 的正常执行
    用法  panic("请设置 ZHIPU_API_KEY 环境变量") — 仅用于不可恢复的初始化错误

  (p *Type) Method()
    说明  指针接收者方法
    用法  func (p *OpenAIProvider) Generate(...) (*schema.Message, error)

── 变量与指针 ──
  :=
    说明  短变量声明
    用法  apiKey := os.Getenv("ZHIPU_API_KEY")

  var
    说明  显式变量声明
    用法  var openaiMsgs []openai.ChatCompletionMessageParamUnion

  &Struct{}
    说明  创建结构体并取地址
    用法  return &OpenAIProvider{client: openai.NewClient(...), model: model}

  *Type 指针
    说明  指针类型，SDK 用 *string 表示可选字符串字段
    用法  openai.String(msg.Content) — 辅助函数返回 *string

── 复合类型 ──
  []Type
    说明  切片
    用法  []openai.ChatCompletionMessageParamUnion, []openai.ChatCompletionToolUnionParam

  map[string]interface{}
    说明  string 到任意类型的映射
    用法  toolDef.InputSchema.(map[string]interface{}) — 类型断言为 map 后提取字段

  shared.FunctionParameters
    说明  OpenAI SDK 定义的 map 别名，本质是 map[string]interface{}
    用法  shared.FunctionParameters(m) — 将 map 转为 SDK 要求类型

── 控制流 ──
  switch/case
    说明  多路分支，每个 case 默认自带 break
    用法  switch msg.Role { case schema.RoleSystem: ... }

  for range {}
    说明  遍历切片
    用法  for _, msg := range msgs { ... }

  for _, v := range
    说明  遍历切片，只取 value
    用法  for _, tc := range choice.ToolCalls { ... }

  if/else
    说明  条件分支
    用法  if msg.ToolCallID != "" { ... } else { ... }

  if _, ok := m[k]; ok
    说明  map 安全读取 + 类型断言组合
    用法  if m, ok := toolDef.InputSchema.(map[string]interface{}); ok { ... }

── 错误处理 ──
  fmt.Errorf + %w
    说明  错误包装
    用法  fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)

  len()
    说明  返回集合长度
    用法  len(resp.Choices) == 0 — 检查 API 是否返回了有效 Choices

── 格式化 ──
  json.Marshal / json.Unmarshal
    说明  JSON 序列化/反序列化
    用法  json.Unmarshal(b, &params) — 将 JSON 字节反序列化到 shared.FunctionParameters

  string([]byte)
    说明  []byte 到 string 显式转换
    用法  string(tc.Arguments) — ToolCall 的 Arguments 是 json.RawMessage（底层 []byte）

── 类型转换 ──
  类型断言 (Type Assertion)
    说明  将 interface{} 值转为具体类型，ok 模式安全转换
    用法  m, ok := toolDef.InputSchema.(map[string]interface{}) — 失败时 ok=false，不会 panic

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 访问其他包导出的标识符
    用法  openai.Client, openai.ChatCompletionNewParams, shared.FunctionDefinitionParam, schema.Message
*/

// 定义 OpenAIProvider 结构体，封装 OpenAI SDK 客户端和模型名
type OpenAIProvider struct {
	client openai.Client // 值类型，非指针
	model  string
}

// 创建智谱 OpenAI 兼容接口的 Provider，通过环境变量获取 API Key
func NewZhipuOpenAIProvider(model string) *OpenAIProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}
	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

// Generate 将内部消息格式转为 OpenAI API 格式发送请求，再将响应转回内部格式
func (p *OpenAIProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var openaiMsgs []openai.ChatCompletionMessageParamUnion

	// 将内部消息逐条转换为 OpenAI SDK 的消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			openaiMsgs = append(openaiMsgs, openai.SystemMessage(msg.Content))

		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 带 ToolCallID 的消息 = 工具执行结果，ToolMessage 给模型回传结果
				openaiMsgs = append(openaiMsgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
			} else {
				openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))
			}
		case schema.RoleAssistant:
			astParam := openai.ChatCompletionAssistantMessageParam{}

			if msg.Content != "" {
				astParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				}
			}

			// 将 ToolCall 列表转为 OpenAI SDK 的工具调用格式
			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
				for _, tc := range msg.ToolCalls {
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID:   tc.ID,
							Type: "function",
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      tc.Name,
								Arguments: string(tc.Arguments),
							},
						},
					})
				}
				astParam.ToolCalls = toolCalls
			}

			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &astParam,
			})
		}
	}

	// 将内部工具定义转为 OpenAI SDK 的 Tool 格式
	var openaiTools []openai.ChatCompletionToolUnionParam
	for _, toolDef := range availableTools {
		var params shared.FunctionParameters
		if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
			params = shared.FunctionParameters(m)
		} else {
			// fallback：类型断言失败时用 JSON 往返兜底转换
			b, _ := json.Marshal(toolDef.InputSchema)
			_ = json.Unmarshal(b, &params)
		}

		openaiTools = append(openaiTools, openai.ChatCompletionFunctionTool(
			shared.FunctionDefinitionParam{
				Name:        toolDef.Name,
				Description: openai.String(toolDef.Description),
				Parameters:  params,
			},
		))
	}

	// 构造 API 请求参数
	params := openai.ChatCompletionNewParams{
		Model:    p.model,
		Messages: openaiMsgs,
	}
	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	// 调用 OpenAI API
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回了空的 Choices")
	}

	// 将 OpenAI 响应转为内部消息格式
	choice := resp.Choices[0].Message
	resultMsg := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Content,
	}

	// 提取工具调用并转为内部 ToolCall 格式
	for _, tc := range choice.ToolCalls {
		if tc.Type == "function" {
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: []byte(tc.Function.Arguments),
			})
		}
	}

	return resultMsg, nil
}
