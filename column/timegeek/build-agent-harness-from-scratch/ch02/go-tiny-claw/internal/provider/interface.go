package provider

import (
	"context"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

/*
Go 语法速查:

── 类型系统 ──
  interface
    说明  定义一组方法签名，实现是隐式的，无需 implements 关键字
    用法  用 type Xxx interface { ... } 声明。任意类型只要实现了接口的所有方法就自动满足该接口，
          编译器在赋值或传参时检查类型是否实现了接口

── 函数与方法 ──
  func Name(p T) Ret
    说明  函数签名，参数名在前类型在后。相同类型的相邻参数可合并写成 a, b int
    用法  func 关键字开头，参数列表后接返回值类型。Generate 方法接收多个参数，返回消息指针和错误

  多返回值
    说明  Go 原生支持返回多个值，常见模式是 (result, error)
    用法  返回值列表中用逗号分隔类型；调用方必须处理返回的 error，否则用 _ 显式丢弃

── 变量与指针 ──
  *Type
    说明  指针类型，& 取变量地址得指针，* 解引用指针取值
    用法  返回 *schema.Message 而非 schema.Message 可避免拷贝，且允许返回 nil。Go 访问指针字段
          直接用 p.Field，无需 ->

── 复合类型 ──
  []Type
    说明  切片，长度可变的动态数组
    用法  函数参数中 []schema.Message 表示接收一个 Message 切片，可遍历、追加

── 上下文 ──
  context.Context
    说明  携带超时、取消信号和请求级别值的上下文，Go 惯例作为函数第一个参数
    用法  从调用方传入，向下传递；不存到 struct 中。入口处用 context.Background() 创建根 ctx

── 包管理 ──
  包名.类型名
    说明  通过 包名.类型名 访问其他包导出的标识符，首字母大写的才是导出的
    用法  schema.Message 指 schema 包下的 Message 类型。import 路径最后一段就是包名

  包名.导出常量
    说明  访问其他包导出的常量，同样遵循首字母大写=导出的规则
    用法  如 schema.RoleUser — RoleUser 是 schema 包导出的常量
*/

// 定义 LLMProvider 接口，约束大模型交互必须实现的方法
type LLMProvider interface {
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
