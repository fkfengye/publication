/*
Go 语法速查:

── 结构体 ──
  type Struct struct { ... }
    说明  字段集合。首字母大写=导出，小写=私有
    用法  type EditFileTool struct { workDir string }

── 构造函数 ──
  func NewXxx() *Xxx
    说明  构造函数，返回类型指针
    用法  func NewEditFileTool(workDir string) *EditFileTool { return &EditFileTool{...} }

── 方法接收者 ──
  (t *EditFileTool) Method()
    说明  指针接收者方法
    用法  func (t *EditFileTool) Name() string { return "edit_file" }

── JSON Tag ──
  `json:"field_name"`
    说明  控制结构体字段序列化时的 JSON 键名
    用法  Path string `json:"path"`; OldText string `json:"old_text"`; NewText string `json:"new_text"`

── 文件读取 ──
  os.ReadFile
    说明  读取整个文件内容到内存，返回字节切片
    用法  contentBytes, err := os.ReadFile(fullPath)

── 文件写入 ──
  os.WriteFile
    说明  写入内容到文件
    用法  os.WriteFile(fullPath, []byte(newContent), 0644)

── 字符串替换 ──
  strings.Replace / strings.ReplaceAll
    说明  Replace 替换前 N 个，ReplaceAll 替换所有匹配项
    用法  strings.ReplaceAll(s, old, new)

── 字符串计数 ──
  strings.Count
    说明  统计子串出现次数
    用法  count := strings.Count(s, oldText)

── 字符串分割 ──
  strings.Split / strings.TrimSpace
    说明  Split 按分隔符分割，TrimSpace 去除首尾空白
    用法  lines := strings.Split(content, "\n"); trimmed := strings.TrimSpace(line)

── 字符串包含 ──
  strings.Contains
    说明  判断字符串是否包含子串
    用法  if strings.Contains(s, substr) { ... }
*/

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourname/go-tiny-claw/internal/schema"
)

// EditFileTool 对现有文件进行局部字符串替换
type EditFileTool struct {
	workDir string
}

// NewEditFileTool 构造函数，创建 EditFileTool 实例
func NewEditFileTool(workDir string) *EditFileTool {
	return &EditFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Definition 返回工具定义，包含工具名称、描述和参数规范
func (t *EditFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "对现有文件进行局部的字符串替换。这比重写整个文件更安全、更快速。请提供足够的 old_text 上下文以确保匹配的唯一性。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要修改的文件路径",
				},
				"old_text": map[string]interface{}{
					"type":        "string",
					"description": "文件中原有的文本。必须包含足够的上下文，以确保在文件中的唯一性。",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "要替换成的新文本",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
	}
}

// editFileArgs 定义 edit_file 工具的输入参数结构
type editFileArgs struct {
	Path    string `json:"path"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

// Execute 对文件进行局部字符串替换；先读 → 模糊匹配 → 写回，三步必须全部成功
func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 1) 反序列化模型传入的 JSON 参数（editFileArgs{Path, OldText, NewText}）
	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 2) 相对路径 → 绝对路径
	fullPath := filepath.Join(t.workDir, input.Path)

	// 3) 先读后改——必须先拿到原文，才能交给 fuzzyReplace 做模糊匹配
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败，请确认路径是否正确: %w", err)
	}
	originalContent := string(contentBytes)

	// 4) 调用四层模糊匹配得到新内容；任何一层 0 处或 >1 处都会返回 error
	newContent, err := fuzzyReplace(originalContent, input.OldText, input.NewText)
	if err != nil {
		return "", err
	}

	// 5) 把新内容原子写回（O_TRUNC + Write），失败时原文未受影响
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("写回文件失败: %w", err)
	}

	return fmt.Sprintf("✅ 成功修改文件: %s", input.Path), nil
}

/*
fuzzyReplace 是本工具的核心算法。它在 originalContent 中查找 oldText 并替换为 newText，
采用四层降级匹配策略：每一层都比上一层更宽松，能容忍更多格式差异。

  L1 精确匹配
    适用场景：模型给出的 oldText 与文件内容字节级完全一致
    行为：唯一匹配则替换；多处匹配则报错（防止误改）

  L2 换行符归一化
    适用场景：跨平台编辑——文件含 \r\n（Windows）而模型给的 oldText 是 \n（Linux 风格）
    行为：把 \r\n 归一为 \n 后再匹配

  L3 行尾空白忽略
    适用场景：模型输出末尾多/少空格，或文件某些行尾被自动 trim 过
    行为：对 oldText 做 TrimSpace 后匹配（不修改原 content，只对查询串宽松）

  L4 逐行去缩进滑动窗口
    适用场景：缩进 / Tab-空格混用导致 L1-L3 全部失败
    行为：委托 lineByLineReplace，把 oldText 拆行、每行 TrimSpace，
          在 content 中按行滑动窗口匹配

所有层级都强制"唯一性"——0 处或多处都不替换，要求模型补充上下文代码。
返回值：(newContent, nil) 或 ("", error)。
*/
func fuzzyReplace(originalContent, oldText, newText string) (string, error) {
	// L1: 精确匹配
	count := strings.Count(originalContent, oldText)
	if count == 1 {
		return strings.Replace(originalContent, oldText, newText, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text 匹配到了 %d 处，请提供更多的上下文代码以确保唯一性", count)
	}

	// L2: 换行符归一化（同时归一 content 和 oldText）
	normalizedContent := strings.ReplaceAll(originalContent, "\r\n", "\n")
	normalizedOld := strings.ReplaceAll(oldText, "\r\n", "\n")

	count = strings.Count(normalizedContent, normalizedOld)
	if count == 1 {
		return strings.Replace(normalizedContent, normalizedOld, newText, 1), nil
	}

	// L3: Trim Space 匹配（空字符串保护，避免把空 oldText 误判为全局匹配）
	trimmedOld := strings.TrimSpace(normalizedOld)
	if trimmedOld != "" {
		count = strings.Count(normalizedContent, trimmedOld)
		if count == 1 {
			return strings.Replace(normalizedContent, trimmedOld, newText, 1), nil
		}
	}

	// L4: 逐行去缩进匹配（兜底层）
	return lineByLineReplace(normalizedContent, normalizedOld, newText)
}

/*
lineByLineReplace 是模糊匹配的兜底层（L4）。

核心思想：把"多行文本匹配"问题降维为"行序列匹配"问题——
  1. 把 oldText 按 \n 拆成行数组 oldLines
  2. 把每行做 TrimSpace（去掉行首缩进和行尾空白）
  3. 把 content 拆成 contentLines
  4. 用滑动窗口在 contentLines 上找连续 len(oldLines) 行全部 TrimSpace 相等的位置
  5. 找到唯一位置时，用 newText 整段替换原文的对应行范围

设计权衡：牺牲严格相等换跨编辑器 / 跨缩进风格的容错。
代价是必须保证 oldText 内不存在"看起来一样但语义不同"的行（如相同代码段重复出现）。
*/
func lineByLineReplace(content, oldText, newText string) (string, error) {
	// 1) 把 content 和 oldText 都按行拆开；oldText 还要做整体 TrimSpace 去掉首尾空行
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(strings.TrimSpace(oldText), "\n")

	// 2) 防御：oldText 为空，或 content 比 oldText 还短，根本不可能匹配
	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return "", fmt.Errorf("找不到该代码片段")
	}

	// 3) 去除旧文本每行的首尾空白，让缩进差异不再影响匹配
	for i := range oldLines {
		oldLines[i] = strings.TrimSpace(oldLines[i])
	}

	matchCount := 0
	matchStartIndex := -1
	matchEndIndex := -1

	// 4) 滑动窗口：起点 i 从 0 一直滑到 contentLines - oldLines
	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		isMatch := true
		for j := 0; j < len(oldLines); j++ {
			// 任意一行 TrimSpace 后不等，就不是匹配点
			if strings.TrimSpace(contentLines[i+j]) != oldLines[j] {
				isMatch = false
				break
			}
		}

		if isMatch {
			matchCount++
			matchStartIndex = i
			matchEndIndex = i + len(oldLines)
		}
	}

	// 5) 0 处匹配：旧文本不在文件中；>1 处匹配：旧文本重复出现，必须消除歧义
	if matchCount == 0 {
		return "", fmt.Errorf("在文件中未找到 old_text，请检查内容和缩进")
	}
	if matchCount > 1 {
		return "", fmt.Errorf("模糊匹配到了 %d 处代码，请提供更多上下文以定位", matchCount)
	}

	// 6) 唯一匹配时执行替换：[匹配前] + newText + [匹配后]
	var newContentLines []string
	newContentLines = append(newContentLines, contentLines[:matchStartIndex]...)
	newContentLines = append(newContentLines, newText)
	newContentLines = append(newContentLines, contentLines[matchEndIndex:]...)

	return strings.Join(newContentLines, "\n"), nil
}
