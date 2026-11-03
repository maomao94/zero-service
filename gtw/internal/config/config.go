package config

import (
	"zero-service/common/gtwx"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	//gateway.GatewayConf
	rest.RestConf
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string            `json:",optional"`
		ClaimMapping     map[string]string `json:",optional"`
	}
	ZeroRpcConf  zrpc.RpcClientConf
	FileRpcConf  zrpc.RpcClientConf
	AdminRpcConf zrpc.RpcClientConf
	NfsRootPath  string
	DownloadUrl  string
	SwaggerPath  string `json:",omitempty"`
	// Error 控制网关错误折叠。默认 Enabled=false，统一透传。
	Error struct {
		Fold gtwx.FoldConfig `json:",optional"`
	} `json:",optional"`
}
