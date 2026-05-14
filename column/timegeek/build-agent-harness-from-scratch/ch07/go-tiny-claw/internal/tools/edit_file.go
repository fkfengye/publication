/*
Go 语法速查:

── 字符串操作 ──
  strings.Count / strings.Replace / strings.ReplaceAll
    说明  Go 标准库 strings 包提供丰富的字符串处理函数
    用法  strings.Count(originalContent, oldText) —
          统计子串出现次数。strings.Replace(s, old, new, n) —
          替换前 n 个匹配，n<0 表示全部替换

  strings.TrimSpace
    说明  去除字符串首尾空白字符
    用法  strings.TrimSpace(normalizedOld) — 空白字符包括空格、制表符、换行等

  strings.Split / strings.Join
    说明  按分隔符切分和合并字符串
    用法  strings.Split(content, "\n") 按换行切分；
          strings.Join(newContentLines, "\n") 按换行合并

── 文件操作 ──
  os.ReadFile / os.WriteFile
    说明  Go 1.16+ 推荐的读写文件方式，一次性读取/写入全部内容
    用法  contentBytes, err := os.ReadFile(fullPath) — 返回 []byte。
          os.WriteFile(fullPath, []byte(newContent), 0644) —
          0644 是文件权限：所有者可读写，组和其他用户只读

  filepath.Join
    说明  智能拼接路径，自动处理分隔符
    用法  filepath.Join(t.workDir, input.Path) —
          自动在 workDir 和 Path 之间添加 / 或 \

── 错误处理 ──
  返回 error vs 返回 (string, error)
    说明  Execute 方法返回 (string, error)，错误时 string 为空，error 填充
    用法  return "", fmt.Errorf("参数解析失败: %w", err) —
          约定：出错时返回空字符串 + 错误；成功时返回输出 + nil

── 多层匹配 ──
  L1/L2/L3/L4 分层匹配策略
    说明  fuzzyReplace 采用四级降级匹配策略，逐层放宽匹配条件
    用法  精确匹配 → 归一化换行 → 去首尾空白 → 逐行去缩进匹配。
          每层若找到唯一匹配则立即返回，否则进入下一层或报错
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

// EditFileTool 提供精确的字符串替换能力，用于局部修改文件
// 这是 ch07 新增的工具，比整文件写入更安全、更精准
type EditFileTool struct {
	workDir string // 工作目录，所有路径基于此解析
}

// NewEditFileTool 创建编辑文件工具实例
func NewEditFileTool(workDir string) *EditFileTool {
	return &EditFileTool{workDir: workDir}
}

// Name 返回工具名称
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Definition 返回工具的 JSON Schema 定义，描述 path/old_text/new_text 三个参数
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

// editFileArgs 用于解析工具调用中的 JSON 参数
type editFileArgs struct {
	Path    string `json:"path"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

// Execute 执行文件编辑操作：读取文件 → 模糊匹配替换 → 写回文件
func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// 解析模型传入的 JSON 参数
	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 拼接完整路径
	fullPath := filepath.Join(t.workDir, input.Path)

	// 读取文件原始内容
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败，请确认路径是否正确: %w", err)
	}
	originalContent := string(contentBytes)

	// 执行模糊匹配替换（四级匹配策略）
	newContent, err := fuzzyReplace(originalContent, input.OldText, input.NewText)
	if err != nil {
		return "", err
	}

	// 写回文件
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("写回文件失败: %w", err)
	}

	return fmt.Sprintf("✅ 成功修改文件: %s", input.Path), nil
}

// fuzzyReplace 实现四级降级匹配策略，逐层放宽匹配条件以容忍缩进/换行差异
func fuzzyReplace(originalContent, oldText, newText string) (string, error) {
	// L1: 精确匹配 -> 效率最高，优先尝试
	count := strings.Count(originalContent, oldText)
	if count == 1 {
		return strings.Replace(originalContent, oldText, newText, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text 匹配到了 %d 处，请提供更多的上下文代码以确保唯一性", count)
	}

	// L2: 换行符归一化（兼容 Windows \r\n 与 Unix \n）
	normalizedContent := strings.ReplaceAll(originalContent, "\r\n", "\n")
	normalizedOld := strings.ReplaceAll(oldText, "\r\n", "\n")

	count = strings.Count(normalizedContent, normalizedOld)
	if count == 1 {
		return strings.Replace(normalizedContent, normalizedOld, newText, 1), nil
	}

	// L3: 去除首尾空白后匹配
	trimmedOld := strings.TrimSpace(normalizedOld)
	if trimmedOld != "" {
		count = strings.Count(normalizedContent, trimmedOld)
		if count == 1 {
			return strings.Replace(normalizedContent, trimmedOld, newText, 1), nil
		}
	}

	// L4: 逐行去缩进匹配（容错能力最强）
	return lineByLineReplace(normalizedContent, normalizedOld, newText)
}

// lineByLineReplace 逐行去除缩进后匹配，用于处理代码缩进不一致的情况
func lineByLineReplace(content, oldText, newText string) (string, error) {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(strings.TrimSpace(oldText), "\n")

	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return "", fmt.Errorf("找不到该代码片段")
	}

	// 去除每行 oldText 的前后空白
	for i := range oldLines {
		oldLines[i] = strings.TrimSpace(oldLines[i])
	}

	matchCount := 0
	matchStartIndex := -1
	matchEndIndex := -1

	// 滑动窗口匹配：在 contentLines 中寻找与 oldLines 逐行（去缩进后）匹配的区间
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

	// 命中替换：将匹配区间替换为 newText
	var newContentLines []string
	newContentLines = append(newContentLines, contentLines[:matchStartIndex]...)
	newContentLines = append(newContentLines, newText)
	newContentLines = append(newContentLines, contentLines[matchEndIndex:]...)

	return strings.Join(newContentLines, "\n"), nil
}
