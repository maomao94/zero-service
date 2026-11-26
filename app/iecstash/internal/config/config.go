package config

import (
	"time"
	"zero-service/common/configx"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaASDUConfig configx.KafkaConsumerConf
	NacosConfig     nacosx.ServiceConfig `json:",optional"`
	StreamEventConf zrpc.RpcClientConf
	DB              struct {
		DataSource string `json:",optional"`
	} `json:",optional"`
	PushAsduChunkBytes int           `json:",default=1048576"` // 1M
	GracePeriod        time.Duration `json:",default=10s"`
}
