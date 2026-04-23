package config

import (
	"time"

	"zero-service/common/mqttx"

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
	MqttConfig mqttx.MqttConfig
	AckTimeout time.Duration `json:",default=5s"`
	PendingTTL time.Duration `json:",default=30s"`
}
