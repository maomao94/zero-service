package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	Model      ModelConfig      `json:"model"`
	Memory     MemoryConfig     `json:"memory"`
	Tools      ToolsConfig      `json:"tools"`
	Skills     SkillsConfig     `json:"skills"`
	Agent      AgentConfig      `json:"agent"`
	Checkpoint CheckpointConfig `json:"checkpoint"`
	Metrics    MetricsConfig    `json:"metrics"`
	Limit      LimitConfig      `json:"limit"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	PoolMaxIdle int           `json:"poolMaxIdle"`
	PoolMaxLive time.Duration `json:"poolMaxLive"`
}

// CheckpointConfig 检查点配置
type CheckpointConfig struct {
	Dir string `json:"dir"`
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled bool `json:"enabled"`
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
	Enabled        bool          `json:"enabled"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"maxRetries"`
	MaxConcurrency int           `json:"maxConcurrency"`
}

// SkillsConfig Skills 配置
type SkillsConfig struct {
	Dir     string `json:"dir"`
	Enabled bool   `json:"enabled"`
}

// LimitConfig 限流配置
type LimitConfig struct {
	MaxConcurrency int           `json:"maxConcurrency"` // 最大并发数
	RateLimit      int           `json:"rateLimit"`      // 每秒请求限制
	RequestTimeout time.Duration `json:"requestTimeout"` // 请求超时时间
}
