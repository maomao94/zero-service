package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/ossx/osssconfig"
)

type Config struct {
	zrpc.RpcServerConf

	DB struct {
		DataSource string
	}

	Cache cache.CacheConf
	Oss   osssconfig.OssConf
}
