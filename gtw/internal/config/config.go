package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
	}
	ZeroRpcConf  zrpc.RpcClientConf
	FileRpcConf  zrpc.RpcClientConf
	AdminRpcConf zrpc.RpcClientConf
	NfsRootPath  string
	DownloadUrl  string
}
