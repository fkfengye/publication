/*
Go 语法速查:

── 接口定义 ──
  type Name interface { Method(params) return }
    说明  接口定义一组方法签名，Go 的接口是隐式实现
    用法  type BaseTool interface { Name() string; Definition() schema.ToolDefinition } —
          不需要 implements 关键字，只要类型拥有全部方法即自动满足接口

  type Registry interface { ... }
    说明  接口即约定，面向接口编程可以解耦调用方与实现方
    用法  engine.AgentEngine 持有 Registry 接口字段，可注入任意实现

── 结构体（实现类型）──
  type registryImpl struct { tools map[string]BaseTool }
    说明  小写字母开头 = 包私有，外部不可直接引用
    用法  NewRegistry() 返回 Registry 接口类型，对外隐藏实现细节

── 工厂函数 ──
  func NewRegistry() Registry
    说明  返回接口而非具体类型，调用方只依赖接口不依赖实现
    用法  registry := tools.NewRegistry() — registry 的类型是 Registry 接口

── map ──
  make(map[string]BaseTool)
    说明  创建 map 的推荐方式，分配内存并初始化
    用法  make(map[string]BaseTool) — map 是引用类型，零值为 nil，
          向 nil map 写入会 panic。make 保证返回的 map 已初始化

── 多重返回值 ──
  value, exists := map[key]
    说明  Go map 读取返回两个值：值和存在标志
    用法  tool, exists := r.tools[call.Name] — exists 为 bool，
          用于判断 key 是否存在。若不存在则 value 为零值

── 格式化 ──
  fmt.Sprintf
    说明  格式化字符串但不输出，返回格式化后的字符串
    用法  errMsg := fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name) —
          与 fmt.Printf 类似但返回字符串供 return 使用

── 日志 ──
  log.Printf
    说明  输出带时间戳的日志
    用法  log.Printf("[Registry] 成功挂载工具: %s\n", name) —
          log 包自动在行尾加换行？不，需要手动写 \n
*/

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// BaseTool 定义单个工具的基本接口：获取名称、定义和执行方法
type BaseTool interface {
	Name() string                                                      // 工具名称，供模型引用
	Definition() schema.ToolDefinition                                 // 工具定义（名称+描述+参数 Schema）
	Execute(ctx context.Context, args json.RawMessage) (string, error) // 执行工具，返回输出文本
}

// Registry 定义工具注册表接口：注册、获取定义、执行工具
type Registry interface {
	Register(tool BaseTool)                                              // 注册一个工具到注册表
	GetAvailableTools() []schema.ToolDefinition                          // 获取所有可用工具的定义列表
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult // 根据 ToolCall 执行对应工具
}

// registryImpl 是 Registry 接口的内部实现，包私有
type registryImpl struct {
	tools map[string]BaseTool // 工具名称 → 工具实例的映射表
}

// NewRegistry 创建并初始化一个空的工具注册表
func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTool), // 初始化 map 防止 nil map 写入 panic
	}
}

// Register 注册工具：同名工具会被覆盖并记录警告
func (r *registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[Warning] 工具 '%s' 已经被注册，将被覆盖。\n", name)
	}
	r.tools[name] = tool
	log.Printf("[Registry] 成功挂载工具: %s\n", name)
}

// GetAvailableTools 收集所有已注册工具的定义列表，供模型了解可用工具
func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var defs []schema.ToolDefinition
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

// Execute 根据 ToolCall 查找并执行对应的工具，返回标准化的 ToolResult
func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	// 查找工具：根据名称在注册表中检索
	tool, exists := r.tools[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	// 执行工具调用
	output, err := tool.Execute(ctx, call.Arguments)

	if err != nil {
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Error executing %s: %v", call.Name, err),
			IsError:    true,
		}
	}

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
