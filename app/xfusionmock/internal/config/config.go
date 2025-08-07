package config

import (
	"zero-service/common/configx"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaTestConfig   kq.KqConf
	KafkaPointConfig  configx.KqConfig
	KafkaAlarmConfig  configx.KqConfig
	KafkaEventConfig  configx.KqConfig
	KafkaTerminalBind configx.KqConfig
	PushCron          string
	PushCronPoint     string
	TerminalBind      map[string]string
	TerminalList      []string
}
