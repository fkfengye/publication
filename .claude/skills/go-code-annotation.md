---
name: go-code-annotation
description: Use when adding comments to Go source files, annotating Go code for beginners, or generating tutorial-style Go code with syntax explanations — applies a three-tier annotation system (file-level syntax reference, block-level functional description, logic-level explanation)
---

# Go Code Annotation

## Overview

为 Go 代码添加结构化注释的技能。核心原则：**文件顶部语法速查 + 代码块功能描述 + 关键逻辑说明**，三层注释体系让 Go 初学者快速理解代码。

## When to Use

- 用户要求为 `.go` 文件添加注释或解释代码
- 编写面向 Go 初学者的教程/示例代码
- 为现有 Go 项目生成带注释的可读版本

**不适用场景：** 生产代码注释（应遵循项目自身注释规范）、非 Go 语言代码

## 注释层级

| 层级 | 内容 | 格式 | 位置 |
|------|------|------|------|
| 文件顶部 | 该文件用到的 Go 语法速查 | `/* ... */` 多行块 | import 语句之后、代码定义之前 |
| 代码块 | 类型/函数/结构体功能描述 | `// 单行` | 定义语句上一行 |
| 关键逻辑 | 业务逻辑说明 | `// 单行` | 逻辑语句上一行 |

## 核心规范

### 1. 文件顶部语法速查块

每个 `.go` 文件顶部添加 `/* Go 语法速查 */` 块，按**分类分组**列出该文件用到的所有语法点。每个语法点包含**说明**（是什么）和**用法**（怎么用）。

**格式规则：**
- 用 `── 分类名 ──` 作为分组标题，与上方内容空一行
- 语法关键字顶格写（2 空格缩进），下方缩进 4 空格分别写 `说明` 和 `用法`
- `说明` 讲概念（是什么），`用法` 讲如何使用（怎么用、何时用、注意事项）
- 用法以文字解释为主，关键位置附短代码片段辅助理解，**禁止只贴代码不解释**
- 说明和用法各自独立一行，不可合并为一行
- 组之间空一行，同组内不同语法点之间空一行
- 只列该文件实际用到的语法点，不罗列无关项
- 代码片段优先取自当前文件实际代码，让读者能对照下方业务代码学习

```go
/*
Go 语法速查:

── 类型系统 ──
  interface
    说明  定义一组方法签名，不包含实现；实现是隐式的，无需 implements 关键字
    用法  用 type Xxx interface { ... } 声明，内部写方法签名。任意类型只要实现了接口的所有方法，
          就自动满足该接口，编译器在赋值时检查。Go 推崇小接口（1-3 个方法），用组合代替大而全

  struct
    说明  字段集合，字段名首字母大写=导出（包外可见），小写=私有（仅包内可见）
    用法  type Name struct { Field Type } 定义，用 Name{Field: value} 构造实例。
          字段可加 json tag 控制序列化：`json:"field_name,omitempty"`

  type Xxx T
    说明  基于底层类型 T 创建新类型，拥有独立的方法集
    用法  type UserID int64 — UserID 是独立类型，不能直接与 int64 运算，需显式转换。
          常用于给基础类型附加语义和方法

── 函数与方法 ──
  func Name(p T) Ret
    说明  函数签名，参数名在前、类型在后（与 C/Java 相反）；多返回值用逗号分隔
    用法  func Add(a, b int) int { return a + b }，相同类型参数可合并写为 a, b int

  (r *Type) Method()
    说明  给类型绑定方法，r 称为接收者。指针接收者可修改原始对象，值接收者操作副本
    用法  func (e *Engine) Run() error { ... }，调用方式 e.Run()。需要修改对象状态时用指针接收者

  多返回值
    说明  Go 原生支持返回多个值，常见模式是 (result, error)，调用方必须处理或显式忽略
    用法  func Lookup(k string) (Value, bool) — 第二个返回值通常表示是否成功。
          用 v, ok := m["key"] 接收，不需要的值用 _ 丢弃

── 变量与指针 ──
  :=
    说明  短变量声明，声明 + 赋值一步完成，编译器自动推断类型
    用法  仅在函数内可用，包级别必须用 var。x := 42 等价于 var x int = 42。
          注意 := 左边至少有一个新变量，否则编译报错

  *Type / &
    说明  & 取变量地址得指针，* 解引用指针取值。指针避免大对象拷贝，允许跨函数修改
    用法  p := &Engine{} 创建指针；*p 取出指向的值。Go 无指针运算，
          访问字段用 p.Field（自动解引用，无需 ->）

── 复合类型 ──
  []Type
    说明  切片，底层是动态数组。长度可变，是 Go 中最常用的集合类型
    用法  msgs := []Message{{Role: "system"}} 字面量创建；msgs = append(msgs, msg) 追加。
          用 make([]Type, len, cap) 预分配容量可减少扩容开销

  map[K]V
    说明  键值对映射，K 必须是可比较类型（string/int/指针等），V 任意
    用法  m := map[string]int{"a": 1}；m["b"] = 2 写入；v, ok := m["key"] 安全读取，
          ok 为 false 时 key 不存在，v 为 V 的零值

── 上下文 ──
  context.Context
    说明  携带截止时间、取消信号和请求级别值的上下文，Go 惯例作为第一个参数传递
    用法  入口处用 context.Background() 或 context.TODO() 创建根 ctx。
          向下传递用 WithCancel/WithTimeout/WithDeadline 派生，不要存入 struct 中

── 包管理 ──
  包名.类型名
    说明  访问其他包的导出类型，只有首字母大写的内容对外可见
    用法  schema.Message{Role: "user"} — schema 是包名，Message 是导出的结构体。
          import 路径最后一段默认就是包名
*/
```

**常用分类参考：**

| 分类 | 包含内容 |
|------|---------|
| 包与导入 | package、import、导出规则 |
| 类型系统 | type、struct、interface |
| 函数与方法 | func、方法接收者、多返回值、构造函数 |
| 变量与指针 | `:=`、`*Type`、`&`、零值、`_` |
| 复合类型 | 切片、map、数组 |
| 控制流 | for、if、switch、break、continue、defer |
| 错误处理 | error、panic、recover、fmt.Errorf |
| 上下文 | context.Context |
| 并发 | goroutine、channel、select |
| 内置函数 | len、append、make、new |
| 格式化 | json tag、`%` 占位符、反引号 |

### 2. 代码块功能描述

每个代码块（type/struct/func/const 等）单独一行 `//` 备注其功能：

```go
// 定义 LLMProvider 接口，约束大模型交互必须实现的方法
type LLMProvider interface {
    ...
}

// 构造函数，初始化 AgentEngine
func NewAgentEngine(...) *AgentEngine {
    ...
}
```

### 3. 单行注释保持简洁

只写功能描述，不写语法关键字（如 `// 定义结构体` 而非 `// type struct 定义结构体`）：

```go
// 初始化对话历史，包含系统提示和用户输入
contextHistory := []schema.Message{...}

// 如果没有工具调用，表示任务完成，退出循环
if len(responseMsg.ToolCalls) == 0 {
    ...
}
```

### 4. 多行注释统一用 /* */

需要多行描述时，统一使用 `/* ... */` 块注释风格，不使用逐行 `//`：

```go
/*
    这是多行注释的第一行
    这是第二行
    ...
*/
```

## 通用规则

**语法速查块（/* */ 块注释）：**
- 只列该文件实际用到的语法点，不罗列无关项
- 用法以文字解释为主，代码片段为辅，禁止只贴代码不解释
- 说明和用法各自独立一行，不合并

**代码块注释（// 单行）：**
- 只描述"做什么"，不描述"怎么做"（代码本身已清晰表达）
- 不重复语法关键字（写 `// 定义引擎结构体` 而非 `// type struct 定义引擎结构体`）
- 代码保持干净，不逐行备注语法

**通用：**
- 单行注释和代码之间保留一个空格
- 保持注释简洁，避免冗长描述

## Common Mistakes

| 错误 | 正确 |
|------|------|
| `// type struct 定义结构体` — 重复语法关键字 | `// 定义 Agent 引擎核心结构体` |
| 逐行给每个字段加注释 | 仅在结构体定义处加功能描述 |
| 文件顶部逐条写 `//` 单行注释 | 用 `/* Go 语法速查 */` 块统一收纳 |
| 用 `//` 拼接写多行注释 | 用 `/* ... */` 块注释 |
| 用法只贴代码片段不解释 | 说明怎么用、何时用、注意点，代码仅为辅助 |
| 语法点不分组合并在一起 | 按 ── 分类名 ── 分组，组间空行分隔 |
| 说明和用法写成一行 | 各自独立一行，`说明` 和 `用法` 作为行首标签 |
| 罗列文件中未用到的语法点 | 只列该文件实际出现的语法，按需精简 |