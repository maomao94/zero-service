package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig nacosx.ServiceConfig `json:",optional"`
	ExtraFields []string             `json:",optional"`
}
