/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type OpenAIProvider struct { client openai.Client; model string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针，负责初始化结构体
    用法  func NewZhipuOpenAIProvider(model string) *OpenAIProvider { return &OpenAIProvider{...} }

── 环境变量 ──
  os.Getenv
    说明  读取环境变量，未设置返回空字符串
    用法  apiKey := os.Getenv("ZHIPU_API_KEY")，敏感信息通过环境变量注入

── 指针字段 ──
  *Type
    说明  指针类型，值类型包装。避免大对象拷贝，允许跨函数修改
    用法  client openai.Client 是值类型，非指针

── 类型断言 ──
  if v, ok := m[key].(Type); ok { ... }
    说明  从接口类型安全提取具体类型，ok 为 false 表示类型不匹配
    用法  m.(map[string]interface{})，用于解析动态 JSON 结构

── 切片追加 ──
  append(slice, element)
    说明  向切片追加元素，返回新切片。当容量不足时会自动扩容
    用法  openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))

── JSON 处理 ──
  json.RawMessage
    说明  原始 JSON 字节数组，不做解析，用于存储未知结构的 JSON 数据
    用法  Arguments json.RawMessage，保留原始参数供下游工具解析

── 错误处理 ──
  error
    说明  内置接口 type error interface { Error() string }
    用法  return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)，用 %w 包装错误链
*/

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

// OpenAIProvider 使用 OpenAI SDK 与大语言模型交互（兼容智谱 Zhipu API）
type OpenAIProvider struct {
	client openai.Client // 值类型，非指针
	model  string
}

// NewZhipuOpenAIProvider 构造函数，创建智谱 API 的 OpenAI 兼容 Provider
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

// Generate 将内部消息格式转换为 OpenAI 格式，调用模型并解析响应
func (p *OpenAIProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var openaiMsgs []openai.ChatCompletionMessageParamUnion

	// 遍历内部消息格式，转换为 OpenAI 消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			openaiMsgs = append(openaiMsgs, openai.SystemMessage(msg.Content))

		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 工具执行结果作为 ToolMessage 返回
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

			// 处理助手消息中的工具调用请求
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

	// 将工具定义转换为 OpenAI Tools 格式
	var openaiTools []openai.ChatCompletionToolUnionParam
	for _, toolDef := range availableTools {
		var params shared.FunctionParameters
		if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
			params = shared.FunctionParameters(m)
		} else {
			// fallback：JSON 往返转换
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

	params := openai.ChatCompletionNewParams{
		Model:    p.model,
		Messages: openaiMsgs,
	}
	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	// 调用 OpenAI/智谱 API
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回了空的 Choices")
	}

	// 解析响应消息
	choice := resp.Choices[0].Message
	resultMsg := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Content,
	}

	// 提取工具调用请求
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
