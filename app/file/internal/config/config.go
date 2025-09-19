package config

import (
	"zero-service/common/ossx/osssconfig"

	"github.com/zeromicro/go-zero/core/stores/cache"
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

	DB struct {
		DataSource string
	}

	Cache cache.CacheConf
	Oss   osssconfig.OssConf

	ThumbTaskConcurrency int `json:",default=2"`
}
