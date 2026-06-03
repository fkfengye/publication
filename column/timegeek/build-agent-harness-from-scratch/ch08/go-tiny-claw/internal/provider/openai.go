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

// Generate 是 Provider 层的"翻译官"：把内部中立消息格式翻成 OpenAI Chat Completion 协议，
// 调用 API 后再把响应翻回内部 schema.Message。本方法分四步：
//   1) msgs 转换    : schema.Message → openai.ChatCompletionMessageParamUnion
//                     （与 Anthropic 关键差异：system 也在 messages 数组里，不单独提取）
//   2) 工具定义转换: schema.ToolDefinition → openai.ChatCompletionToolUnionParam
//   3) HTTP 调用   : p.client.Chat.Completions.New(ctx, params)
//   4) 响应解析    : resp.Choices[0].Message → schema.Message
func (p *OpenAIProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	// 1) 转换消息历史：内部 schema 格式 → OpenAI ChatCompletion 格式
	var openaiMsgs []openai.ChatCompletionMessageParamUnion

	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			// OpenAI 协议与 Anthropic 关键差异：system 是 messages 数组的一员
			openaiMsgs = append(openaiMsgs, openai.SystemMessage(msg.Content))

		case schema.RoleUser:
			// 区分"普通 user 文本"和"工具结果回调"——通过 ToolCallID 是否非空判断
			if msg.ToolCallID != "" {
				// 工具结果：必须用 ToolMessage 构造函数，自动带 role=tool + tool_call_id
				openaiMsgs = append(openaiMsgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
			} else {
				// 普通用户文本
				openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))
			}
		case schema.RoleAssistant:
			// assistant 消息可同时含"文本回复"和"多个工具调用请求"
			astParam := openai.ChatCompletionAssistantMessageParam{}

			if msg.Content != "" {
				astParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				}
			}

			// 把历史 tool_calls 翻译成 OpenAI 的 function tool_calls
			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
				for _, tc := range msg.ToolCalls {
					// OpenAI 的 Function.Arguments 是 string（不是 json.RawMessage）——直接 string() 强转
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID:   tc.ID,
							Type: "function", // OpenAI 协议固定 "function" 字面量
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

	// 2) 转换工具定义：内部 schema.ToolDefinition → OpenAI ChatCompletion Tool 格式
	var openaiTools []openai.ChatCompletionToolUnionParam
	for _, toolDef := range availableTools {
		var params shared.FunctionParameters
		// OpenAI SDK 用 shared.FunctionParameters 类型（实际是 map[string]any 的别名）
		// 优先做类型断言：成功就直接转换；失败就 JSON 往返一次（兜底）
		if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
			params = shared.FunctionParameters(m)
		} else {
			// fallback：marshal 再 unmarshal 一次，相当于反射克隆
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

	// 3) 构造请求参数
	params := openai.ChatCompletionNewParams{
		Model:    p.model,
		Messages: openaiMsgs,
	}
	// 挂载工具；空切片不发送，避免 SDK 触发校验
	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	// 4) 真正发 HTTP 请求到智谱 Zhipu（BaseURL 已在 NewZhipuOpenAIProvider 中切好）
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)
	}
	// 防御：智谱偶尔可能返回空 Choices（如内容安全拦截），显式报错而不是 panic
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回了空的 Choices")
	}

	// 5) 解析响应：取第 0 个 choice（流式场景下会有多个，非流式只关心第一个）
	choice := resp.Choices[0].Message
	resultMsg := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Content, // OpenAI 单段文本，直接拿
	}

	// 提取工具调用请求；OpenAI 的 tool_call.Type == "function" 是协议固定值
	for _, tc := range choice.ToolCalls {
		if tc.Type == "function" {
			// Arguments 是 string，强转 []byte 得到 json.RawMessage 等价物
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: []byte(tc.Function.Arguments),
			})
		}
	}

	return resultMsg, nil
}
