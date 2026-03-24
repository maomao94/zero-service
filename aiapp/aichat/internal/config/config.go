package config

import (
	"time"

	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/zrpc"
)

type ProviderConfig struct {
	Name     string
	Type     string // "openai_compatible"
	Endpoint string
	ApiKey   string
}

type ModelConfig struct {
	Id                string
	Provider          string
	BackendModel      string
	DisplayName       string `json:",optional"`
	Description       string `json:",optional"`
	MaxTokens         int    `json:",optional,default=8192"`
	SupportsStreaming bool   `json:",optional,default=true"`
}

type Config struct {
	zrpc.RpcServerConf
	StreamTimeout     time.Duration `json:",default=10m"` // 单次流的总时长上限
	StreamIdleTimeout time.Duration `json:",default=90s"` // chunk 间最大空闲时间
	MaxToolRounds     int           `json:",default=10"`  // tool-calling 最大循环轮次
	Providers         []ProviderConfig
	Models            []ModelConfig
	McpServers        []McpServerConfig `json:",optional"` // Deprecated: 使用 Mcpx 替代
	Mcpx              mcpx.Config       `json:",optional"` // MCP 客户端配置
}

type McpServerConfig struct {
	Name     string `json:",optional"`
	Endpoint string // MCP SSE endpoint, e.g. "http://localhost:13003/sse"
}

// GetMcpxConfig 兼容旧配置：如果 Mcpx.Servers 为空但 McpServers 不为空，自动迁移。
func (c Config) GetMcpxConfig() mcpx.Config {
	if len(c.Mcpx.Servers) > 0 {
		return c.Mcpx
	}
	if len(c.McpServers) == 0 {
		return c.Mcpx
	}
	cfg := c.Mcpx
	cfg.Servers = make([]mcpx.ServerConfig, len(c.McpServers))
	for i, s := range c.McpServers {
		cfg.Servers[i] = mcpx.ServerConfig{
			Name:     s.Name,
			Endpoint: s.Endpoint,
		}
	}
	return cfg
}
