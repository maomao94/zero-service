package config

import (
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig nacosx.ServiceConfig `json:",optional"`
	DB          gormx.Config         `json:",optional"`
}
