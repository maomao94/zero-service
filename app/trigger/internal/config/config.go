package config

import (
	"time"

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
	RedisDB int `json:",optional"`
	DB      struct {
		DataSource string
	}
	DisableStmtLog bool          `json:",optional"`
	GracePeriod    time.Duration `json:",default=30s"`
}
