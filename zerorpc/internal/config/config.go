package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ZeroAlarmConf zrpc.RpcClientConf
	JwtAuth       struct {
		AccessSecret string
		AccessExpire int64
	}
	MiniProgram struct {
		AppId  string
		Secret string
	}

	DB struct {
		DataSource string
	}
	Cache cache.CacheConf
}
