/*
Go 语法速查:

── 结构体与值类型 ──
  openai.Client // 值类型，非指针
    说明  openai 新版 SDK 的 Client 是值类型而非指针，可直接持有无需取地址
    用法  type OpenAIProvider struct { client openai.Client } —
          值类型意味着赋值时发生拷贝，通常更安全且无需手动管理生命周期

── 类型转换 ──
  openai.String(msg)
    说明  将 Go 的 string 转为 SDK 定义的 String 类型
    用法  openai.String(toolDef.Description) — 很多 SDK 会定义自己的基本类型封装，
          通过工具函数进行转换

  shared.FunctionParameters(m)
    说明  显式类型转换，将 map 转为 SDK 期望的参数类型
    用法  params = shared.FunctionParameters(m) — 当底层类型相同时，
          Go 允许通过 T(x) 进行显式转换

── 结构体嵌套 ──
  匿名嵌套字段的赋值
    说明  SDK 中大量使用嵌套结构体作为字段类型
    用法  astParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
            OfString: openai.String(msg.Content),
          } — 需要层层构造嵌套结构体，是类型安全的 JSON 构造方式

── 指针构造 ──
  openai.String(...)
    说明  返回指向字符串的指针
    用法  Description: openai.String(toolDef.Description) — SDK 中可选字段往往声明为指针，
          通过工具函数一步完成取地址

── JSON 往返 ──
  Marshal → Unmarshal
    说明  通过 JSON 序列化再反序列化实现类型转换的兜底方案
    用法  b, _ := json.Marshal(toolDef.InputSchema)
          _ = json.Unmarshal(b, &params) — 当直接类型断言失败时，
          通过 JSON 中转将任意结构转为目标类型

── 方法调用 ──
  p.client.Chat.Completions.New(ctx, params)
    说明  OpenAI v3 SDK 的链式调用风格
    用法  p.client.Chat.Completions.New(ctx, params) —
          Chat → Completions → New 三级路径访问 Chat Completion API

── 参数校验 ──
  len(resp.Choices) == 0
    说明  响应完整性检查
    用法  if len(resp.Choices) == 0 { return nil, fmt.Errorf("API 返回了空的 Choices") } —
          在解析响应前检查关键字段，确保后续访问不会 panic
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

// OpenAIProvider 通过 OpenAI SDK 对接智谱 GLM 模型（兼容 OpenAI API 格式）
type OpenAIProvider struct {
	client openai.Client // 值类型，非指针
	model  string
}

// NewZhipuOpenAIProvider 创建 OpenAI 格式的智谱 Provider，从环境变量读取 API Key
func NewZhipuOpenAIProvider(model string) *OpenAIProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}
	// 智谱的 API 端点兼容 OpenAI Chat Completion API 格式
	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

// Generate 实现 LLMProvider 接口：将内部 Message 格式转为 OpenAI SDK 格式并发起调用
func (p *OpenAIProvider) Generate(
	ctx context.Context,
	msgs []schema.Message,
	availableTools []schema.ToolDefinition) (*schema.Message, error) {

	var openaiMsgs []openai.ChatCompletionMessageParamUnion

	// 将内部统一的 Message 切片转换为 OpenAI SDK 的消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			// OpenAI API：system 消息放入 messages 数组，使用 SystemMessage 构造
			openaiMsgs = append(openaiMsgs, openai.SystemMessage(msg.Content))

		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 工具执行结果回传 → 使用 ToolMessage
				openaiMsgs = append(openaiMsgs, openai.ToolMessage(msg.Content, msg.ToolCallID))
			} else {
				// 普通用户文本消息
				openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))
			}

		case schema.RoleAssistant:
			// 构造 Assistant 消息，包含文本内容和工具调用
			astParam := openai.ChatCompletionAssistantMessageParam{}

			if msg.Content != "" {
				astParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				}
			}

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

	// 将内部 ToolDefinition 转换为 OpenAI SDK 的工具参数格式
	var openaiTools []openai.ChatCompletionToolUnionParam
	for _, toolDef := range availableTools {
		var params shared.FunctionParameters
		// 从通用 InputSchema 中提取 FunctionParameters，优先用类型断言
		if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
			params = shared.FunctionParameters(m)
		} else {
			// 兜底：通过 JSON 往返转换
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

	// 组装 API 请求参数
	params := openai.ChatCompletionNewParams{
		Model:    p.model,
		Messages: openaiMsgs,
	}
	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	// 发起 API 调用
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Zhipu API 请求失败: %w", err)
	}

	// 校验响应完整性
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回了空的 Choices")
	}

	// 将 OpenAI SDK 响应转回内部 Message 格式
	choice := resp.Choices[0].Message
	resultMsg := &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Content,
	}

	// 提取工具调用信息
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
