package config

import (
	"zero-service/common/configx"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig        nacosx.ServiceConfig        `json:",optional"`
	KafkaPushConfig    configx.KafkaMultiPushConf  `json:",optional"`
	KafkaConsumeConfig []configx.KafkaConsumerConf `json:",optional"`
	StreamEventConf    zrpc.RpcClientConf          `json:",optional"`
}
