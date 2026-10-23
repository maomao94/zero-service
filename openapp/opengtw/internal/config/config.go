package config

import (
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	// JwtAuth 管理面鉴权配置，复用 gtw 的配置形态。
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string            `json:",optional"`
		ClaimMapping     map[string]string `json:",optional"`
	}
	// DB 存放 open_app / open_app_secret；本地默认 zero 库，生产建议独立 resource 库。
	DB gormx.Config `json:",optional"`
	// OpenApiConf facade/openapi 转发服务。
	OpenApiConf zrpc.RpcClientConf
}
