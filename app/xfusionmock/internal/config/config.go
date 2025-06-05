package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/config"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaTestConfig   kq.KqConf
	KafkaPointConfig  config.KqConfig
	KafkaAlarmConfig  config.KqConfig
	KafkaEventConfig  config.KqConfig
	KafkaTerminalBind config.KqConfig
	PushCron          string
	PushCronPoint     string
}
