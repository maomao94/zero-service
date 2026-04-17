package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config aisolo 服务配置
type Config struct {
	zrpc.RpcServerConf

	Model        ModelConfig        `json:"model"`
	Memory       MemoryConfig       `json:"memory"`
	SessionStore SessionStoreConfig `json:",optional"`
	DB           DBConfig           `json:",optional"`
	Tools        ToolsConfig        `json:"tools"`
	Skills       SkillsConfig       `json:"skills"`
	Agent        AgentConfig        `json:"agent"`
	Checkpoint   CheckpointConfig   `json:"checkpoint"`
	Metrics      MetricsConfig      `json:"metrics"`
	Limit        LimitConfig        `json:"limit"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	PoolMaxIdle int           `json:"poolMaxIdle"`
	PoolMaxLive time.Duration `json:"poolMaxLive"`
}

// CheckpointConfig Agent 中断/恢复的快照存储配置。
// 与 Memory/SessionStore 保持一致的 memory|jsonl|gormx 三后端模型。
type CheckpointConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
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

// MemoryConfig 记忆配置（消息存储）。
type MemoryConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
}

// SessionStoreConfig 会话存储配置。
type SessionStoreConfig struct {
	Type    string `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 存储目录
}

// DBConfig gormx 数据库配置。
type DBConfig struct {
	Enabled    bool   `json:",optional,default=false"`
	DataSource string `json:",optional"`
	LogLevel   string `json:",optional,default=error"`
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
