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

// Execute 对文件进行局部替换
func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	fullPath := filepath.Join(t.workDir, input.Path)

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败，请确认路径是否正确: %w", err)
	}
	originalContent := string(contentBytes)

	newContent, err := fuzzyReplace(originalContent, input.OldText, input.NewText)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("写回文件失败: %w", err)
	}

	return fmt.Sprintf("✅ 成功修改文件: %s", input.Path), nil
}

// fuzzyReplace 模糊替换，支持多种匹配策略
// L1: 精确匹配
// L2: 换行符归一化（\r\n -> \n）
// L3: TrimSpace 匹配
// L4: 逐行去缩进匹配
func fuzzyReplace(originalContent, oldText, newText string) (string, error) {
	// L1: 精确匹配
	count := strings.Count(originalContent, oldText)
	if count == 1 {
		return strings.Replace(originalContent, oldText, newText, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text 匹配到了 %d 处，请提供更多的上下文代码以确保唯一性", count)
	}

	// L2: 换行符归一化
	normalizedContent := strings.ReplaceAll(originalContent, "\r\n", "\n")
	normalizedOld := strings.ReplaceAll(oldText, "\r\n", "\n")

	count = strings.Count(normalizedContent, normalizedOld)
	if count == 1 {
		return strings.Replace(normalizedContent, normalizedOld, newText, 1), nil
	}

	// L3: Trim Space 匹配
	trimmedOld := strings.TrimSpace(normalizedOld)
	if trimmedOld != "" {
		count = strings.Count(normalizedContent, trimmedOld)
		if count == 1 {
			return strings.Replace(normalizedContent, trimmedOld, newText, 1), nil
		}
	}

	// L4: 逐行去缩进匹配
	return lineByLineReplace(normalizedContent, normalizedOld, newText)
}

// lineByLineReplace 逐行匹配并替换，跳过行首缩进差异
func lineByLineReplace(content, oldText, newText string) (string, error) {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(strings.TrimSpace(oldText), "\n")

	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return "", fmt.Errorf("找不到该代码片段")
	}

	// 去除旧文本每行的首尾空白
	for i := range oldLines {
		oldLines[i] = strings.TrimSpace(oldLines[i])
	}

	matchCount := 0
	matchStartIndex := -1
	matchEndIndex := -1

	// 滑动窗口查找匹配位置
	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		isMatch := true
		for j := 0; j < len(oldLines); j++ {
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

	if matchCount == 0 {
		return "", fmt.Errorf("在文件中未找到 old_text，请检查内容和缩进")
	}
	if matchCount > 1 {
		return "", fmt.Errorf("模糊匹配到了 %d 处代码，请提供更多上下文以定位", matchCount)
	}

	// 执行替换
	var newContentLines []string
	newContentLines = append(newContentLines, contentLines[:matchStartIndex]...)
	newContentLines = append(newContentLines, newText)
	newContentLines = append(newContentLines, contentLines[matchEndIndex:]...)

	return strings.Join(newContentLines, "\n"), nil
}
