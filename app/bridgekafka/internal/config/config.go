package config

import (
	"zero-service/common/configx"

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
	KafkaPushConfig    configx.KafkaMultiPushConf  `json:",optional"`
	KafkaConsumeConfig []configx.KafkaConsumerConf `json:",optional"`
	StreamEventConf    zrpc.RpcClientConf          `json:",optional"`
}
