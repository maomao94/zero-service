package config

import (
	"time"

	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig     nacosx.ServiceConfig `json:",optional"`
	RedisDB         int                  `json:",optional"`
	DB              gormx.Config         `json:",optional"`
	GracePeriod     time.Duration        `json:",default=30s"`
	StreamEventConf zrpc.RpcClientConf
	CronTask        CronTaskConfig `json:",optional"`
	// SocketPushConf socketpush 推送服务（可选，未配置时批次确认只记录日志）
	SocketPushConf zrpc.RpcClientConf `json:",optional"`
}

type CronTaskConfig struct {
	MaxDelay time.Duration `json:",default=30m"`
}
