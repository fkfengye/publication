package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

/*
Go 语法速查:

── 类型系统 ──
  type struct
    说明  定义结构体，字段标签 yaml:"xxx" 映射 YAML 文件中的 key
    用法  type Config struct { APIKey string `yaml:"api_key"` }

  struct tag
    说明  结构体字段的元数据，用反引号包裹，`yaml:"api_key"` 表示 YAML 中 key 为 api_key
    用法  APIKey string `yaml:"api_key"`

── 函数与方法 ──
  func Load(path string) (*Config, error)
    说明  从指定路径加载 YAML 配置文件，返回 Config 指针和 error
    用法  cfg, err := config.Load("config.yaml")

  os.ReadFile
    说明  读取文件全部内容到 []byte
    用法  data, err := os.ReadFile(path)

  yaml.Unmarshal
    说明  将 YAML 字节反序列化到结构体
    用法  yaml.Unmarshal(data, &cfg)

── 错误处理 ──
  fmt.Errorf + %w
    说明  错误包装，保留原始错误链
    用法  fmt.Errorf("读取配置文件失败: %w", err)

── 变量与指针 ──
  &Config{}
    说明  创建结构体并取地址返回
    用法  cfg := &Config{} — 声明空配置指针供 Unmarshal 填充
*/

// Config 配置文件结构体，映射 config.yaml 中的配置项
type Config struct {
	APIKey   string `yaml:"api_key"`  // LLM API 密钥
	BaseURL  string `yaml:"base_url"` // LLM API 基础地址
	Model    string `yaml:"model"`    // 模型名
	WorkDir  string `yaml:"work_dir"` // 工作区路径，为空则使用当前目录
	Thinking bool   `yaml:"thinking"` // 是否启用 thinking 模式
}

// Load 从指定路径加载 YAML 配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 环境变量覆盖：若配置文件中未设置 api_key，则回退读取环境变量
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ZHIPU_API_KEY")
	}

	return cfg, nil
}
