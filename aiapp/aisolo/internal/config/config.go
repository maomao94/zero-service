package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	Model  ModelConfig  `json:"model"`
	Memory MemoryConfig `json:"memory"`
	Tools  ToolsConfig  `json:"tools"`
	Skills SkillsConfig `json:"skills"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	APIKey      string  `json:"apiKey"`
	BaseURL     string  `json:"baseURL,omitempty"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	Type string `json:"type"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Enabled bool `json:"enabled"`
}

// SkillsConfig Skills 配置
type SkillsConfig struct {
	Dir     string `json:"dir"`
	Enabled bool   `json:"enabled"`
}
