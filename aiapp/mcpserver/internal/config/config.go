package config

import (
	"github.com/zeromicro/go-zero/mcp"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	mcp.McpConf
	BridgeModbusRpcConf zrpc.RpcClientConf
}
