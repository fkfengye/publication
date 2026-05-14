/*
Go 语法速查:

── 结构体与方法 ──
  type StructName struct { ... }
    说明  定义结构体类型
    用法  type ClaudeProvider struct { client anthropic.Client; model string }
          结构体嵌入 SDK 的 Client，将其作为字段持有而非嵌入继承

  方法定义
    说明  Go 方法就是带接收者的函数
    用法  func (p *ClaudeProvider) Generate(...) — 指针接收者，
          虽然本方法未修改 p 的字段，但保持指针接收者的一致性

── 切片与循环 ──
  for range 切片
    说明  遍历切片元素
    用法  for _, msg := range msgs { ... } — 用 _ 忽略索引

  append 动态扩容
    说明  向切片追加元素
    用法  anthropicMsgs = append(anthropicMsgs, ...) — 切片自动扩容

── switch 语句 ──
  switch xxx { case A: ... case B: ... }
    说明  分支控制，每个 case 默认带 break，不会自动穿透
    用法  switch msg.Role { case schema.RoleSystem: ... case schema.RoleUser: ... } —
          Go 的 switch 不需要写 break，如需穿透使用 fallthrough 关键字

── 类型断言 ──
  m.(map[string]interface{})
    说明  尝试将接口类型转换为具体类型，ok 为是否成功
    用法  if m, ok := toolDef.InputSchema.(map[string]interface{}); ok { ... } —
          如果不判断 ok，失败时会 panic

── map 操作 ──
  map[string]interface{}
    说明  Go 的 map 是引用类型，零值为 nil，使用前需要用 make 初始化
    用法  var inputMap map[string]interface{}; json.Unmarshal(tc.Arguments, &inputMap) —
          虽然 inputMap 是 nil，但 json.Unmarshal 会初始化它

── 指针与值 ──
  & 取地址
    说明  & 获取变量的内存地址
    用法  anthropic.ToolInputSchemaParam{ Properties: properties } —
          字段接收 map[string]any 是值类型，直接赋值即可

── JSON 序列化 ──
  json.Marshal / json.Unmarshal
    说明  序列化（对象→JSON）和反序列化（JSON→对象）
    用法  argsBytes, _ := json.Marshal(block.Input) — 将 map 转回 JSON bytes

── 第三方 SDK ──
  anthropic-sdk-go
    说明  Anthropic 官方 Go SDK，提供 Messages API 调用
    用法  NewToolResultBlock(msg.ToolCallID, msg.Content, false) —
          构造工具结果块，对应 Anthropic Messages API 的 tool_result 类型

── 错误处理 ──
  fmt.Errorf + %w
    说明  包装错误，保留错误链
    用法  fmt.Errorf("Claude/Zhipu API 请求失败: %w", err) —
          %w 只能用于 error 类型的值，调用方可用 errors.Is/As 解包
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

// ClaudeProvider 通过 Anthropic SDK 对接智谱 GLM 模型（兼容 Claude API 格式）
type ClaudeProvider struct {
	client anthropic.Client // Anthropic SDK 客户端
	model  string           // 模型名称，如 "glm-4.5-flash"
}

// NewZhipuClaudeProvider 创建 Claude 格式的智谱 Provider，从环境变量读取 API Key
func NewZhipuClaudeProvider(model string) *ClaudeProvider {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		panic("请设置 ZHIPU_API_KEY 环境变量")
	}
	// 智谱的 API 端点兼容 Claude Messages API 格式
	baseURL := "https://open.bigmodel.cn/api/paas/v4/"
	return &ClaudeProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

// Generate 实现 LLMProvider 接口：将内部 Message 格式转为 Anthropic SDK 格式并发起调用
func (p *ClaudeProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	var anthropicMsgs []anthropic.MessageParam
	var systemPrompt string

	// 将内部统一的 Message 切片转换为 Anthropic SDK 的消息格式
	for _, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			// Claude API：system 提示词单独作为参数，不放入 messages 数组
			systemPrompt = msg.Content

		case schema.RoleUser:
			if msg.ToolCallID != "" {
				// 这是工具执行结果的回传消息 → 构造 ToolResult 块
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			} else {
				// 普通用户文本消息
				anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}

		case schema.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			// 若模型有文本回复，添加 TextBlock
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			// 若模型发起了工具调用，逐一构造 ToolUseBlock
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

	// 将内部 ToolDefinition 转换为 Anthropic SDK 的工具参数格式
	var anthropicTools []anthropic.ToolUnionParam
	for _, toolDef := range availableTools {
		var properties map[string]any
		var required []string

		// 从通用 InputSchema 中提取 Anthropic SDK 需要的 properties 和 required
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

	// 组装 API 请求参数
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMsgs,
	}

	// 如果有 system 提示词，设置到请求中
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	// 如果有可用工具，设置工具列表
	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	// 发起 API 调用
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude/Zhipu API 请求失败: %w", err)
	}

	// 将 Anthropic SDK 响应转回内部 Message 格式
	resultMsg := &schema.Message{
		Role: schema.RoleAssistant,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			// 文本内容块
			resultMsg.Content += block.Text
		case "tool_use":
			// 工具调用块：将 Input 序列化为 JSON bytes 存储
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
