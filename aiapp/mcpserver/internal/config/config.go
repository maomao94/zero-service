package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Auth struct {
		JwtSecrets   []string `json:",optional"`
		ServiceToken string   `json:",optional"`
	}
	Mcp struct {
		Name            string        `json:",default=mcpserver"`
		Version         string        `json:",default=1.0.0"`
		MessageEndpoint string        `json:",default=/message"`
		Cors            []string      `json:",optional"`
		SessionTimeout  time.Duration `json:",default=24h"`
	}
	BridgeModbusRpcConf zrpc.RpcClientConf
}
