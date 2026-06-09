package config

import (
	"github.com/zeromicro/go-queue/kq"
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
	KafkaPushConfig    KafkaPushConfig    `json:",optional"`
	KafkaConsumeConfig kq.KqConf          `json:",optional"`
	StreamEventConf    zrpc.RpcClientConf `json:",optional"`
}

type KafkaPushConfig struct {
	Brokers []string
	Topics  []string
}
