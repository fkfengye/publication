package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  interface
    说明  定义一组方法签名，实现是隐式的。接口越小越灵活
    用法  type BaseTool interface { Name() string; Definition() schema.ToolDefinition; Execute(...) (string, error) }
          — 定义了三个方法，任意类型实现了它们就自动满足 BaseTool 接口

  type struct
    说明  定义结构体，小写类型名为包私有
    用法  type registryImpl struct { tools map[string]BaseTool } — 内部实现，外部不可见

── 变量与指针 ──
  &Struct{...}
    说明  取结构体字面量的地址，返回指针
    用法  return &registryImpl{tools: make(map[string]BaseTool)} — 构造函数中创建并返回指针

  (r *Type) Method()
    说明  指针接收者方法，可修改接收者状态
    用法  func (r *registryImpl) Register(tool BaseTool) — r 指向的 registryImpl 会被修改

  make()
    说明  分配并初始化 map/切片/chan，返回已初始化的值（非零值）
    用法  make(map[string]BaseTool) — 创建空的 map，之后可用 m["key"] = val 写入

── 复合类型 ──
  map[K]V
    说明  键值对映射，K 必须可比较，V 任意类型
    用法  tools map[string]BaseTool — 用工具名做 key，工具实例做 value。读取用 v, ok := m["key"]

  []Type
    说明  切片，动态数组
    用法  var defs []schema.ToolDefinition — 声明空切片；for range 遍历，append 追加

── 控制流 ──
  for range {}
    说明  遍历切片/map，每次迭代返回 key 和 value
    用法  for _, tool := range r.tools { defs = append(defs, tool.Definition()) } — 只取 value

  if _, ok := m[k]; ok
    说明  map 安全读取惯用写法，ok 为 true 表示 key 存在
    用法  if _, exists := r.tools[name]; exists { ... } — 检查工具是否已注册

  if/else
    说明  条件分支
    用法  if err != nil { return errorResult } else { return successResult }

── 错误处理 ──
  fmt.Sprintf
    说明  格式化字符串并返回，不输出到控制台
    用法  fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name) — %s 替换字符串

── 格式化 ──
  log.Printf
    说明  格式化日志输出，支持 %s %d %v 等占位符
    用法  log.Printf("[Registry] 成功挂载工具: %s\n", name)

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 引用其他包导出的类型
    用法  schema.ToolDefinition, schema.ToolResult — schema 包的类型在此处使用
*/

// 定义 BaseTool 接口，所有工具必须实现 Name、Definition、Execute 三个方法
type BaseTool interface {
	Name() string
	Definition() schema.ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// 定义 Registry 接口，管理工具注册、查询和执行
type Registry interface {
	Register(tool BaseTool)
	GetAvailableTools() []schema.ToolDefinition
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}

// 注册表内部实现，用 map 存储工具名到工具实例的映射
type registryImpl struct {
	tools map[string]BaseTool
}

// 构造函数，创建空注册表并初始化 map
func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTool),
	}
}

// 将工具注册到注册表中，同名工具会被覆盖并输出警告
func (r *registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[Warning] 工具 '%s' 已经被注册，将被覆盖。\n", name)
	}
	r.tools[name] = tool
	log.Printf("[Registry] 成功挂载工具: %s\n", name)
}

// 获取所有已注册工具的定义列表，供 LLM Provider 构建请求时使用
func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var defs []schema.ToolDefinition
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

// 根据工具调用信息执行对应工具，返回执行结果
func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	// 查找工具是否存在
	tool, exists := r.tools[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Error: 系统中不存在名为 '%s' 的工具。", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	// 执行工具
	output, err := tool.Execute(ctx, call.Arguments)

	// 执行失败时返回错误信息而非抛出异常，保证主循环不会中断
	if err != nil {
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Error executing %s: %v", call.Name, err),
			IsError:    true,
		}
	}

	// 执行成功
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
