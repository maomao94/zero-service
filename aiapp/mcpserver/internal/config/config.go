package config

import (
	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	mcpx.McpServerConf
	BridgeModbusRpcConf zrpc.RpcClientConf
}

// SkillsConfig skills 目录配置
type SkillsConfig struct {
	Dir        string `json:"dir"`                  // skills 目录路径
	AutoReload bool   `json:"autoReload,omitempty"` // 是否热加载
}
