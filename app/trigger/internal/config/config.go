package config

import (
	"time"

	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	RedisDB         int           `json:",optional"`
	DB              gormx.Config  `json:",optional"`
	GracePeriod     time.Duration `json:",default=30s"`
	StreamEventConf zrpc.RpcClientConf
}
