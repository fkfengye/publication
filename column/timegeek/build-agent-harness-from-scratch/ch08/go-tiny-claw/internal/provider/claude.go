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

// Generate 将内部消息格式转换为 Anthropic 格式，调用模型并解析响应
func (p *ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var anthropicMsgs []anthropic.MessageParam
	var systemPrompt string

	// 遍历内部消息格式，转换为 Anthropic 消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			systemPrompt = msg.Content
		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 工具执行结果作为 ToolResultBlock 返回
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			} else {
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case schema.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			// 处理助手消息中的工具调用请求
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

	// 将工具定义转换为 Anthropic Tools 格式
	var anthropicTools []anthropic.ToolUnionParam
	for _, toolDef := range availableTools {
		var properties map[string]any
		var required []string

		// 从 InputSchema 中提取 properties 和 required
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

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMsgs,
	}

	// 设置系统提示
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	// 设置可用工具
	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	// 调用 Claude/智谱 API
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)
	}

	// 解析响应消息
	resultMsg := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			resultMsg.Content += block.Text
		case "tool_use":
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
