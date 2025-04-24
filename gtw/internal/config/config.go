package config

import (
	"github.com/zeromicro/go-zero/gateway"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	gateway.GatewayConf
	JwtAuth struct {
		AccessSecret string
	}
	ZeroRpcConf  zrpc.RpcClientConf
	FileRpcConf  zrpc.RpcClientConf
	AdminRpcConf zrpc.RpcClientConf
	NfsRootPath  string
	DownloadUrl  string
}
