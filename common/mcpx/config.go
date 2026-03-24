package mcpx

import "time"

const (
	DefaultRefreshInterval = 30 * time.Second
	DefaultConnectTimeout  = 10 * time.Second
	ToolNameSeparator      = "__"
)

type ServerConfig struct {
	Name         string `json:",optional"` // 工具名前缀，为空自动生成 mcp0, mcp1...
	Endpoint     string // MCP Streamable HTTP endpoint
	ServiceToken string `json:",optional"` // 连接级鉴权 token
}

type Config struct {
	Servers         []ServerConfig `json:",optional"`
	RefreshInterval time.Duration  `json:",default=30s"` // 断开后重连间隔 / SDK KeepAlive 间隔
	ConnectTimeout  time.Duration  `json:",default=10s"` // 单次连接超时
}
