package config

import (
	"zero-service/common/configx"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaTestConfig   kq.KqConf
	KafkaPointConfig  configx.KafkaPushConf
	KafkaAlarmConfig  configx.KafkaPushConf
	KafkaEventConfig  configx.KafkaPushConf
	KafkaTerminalBind configx.KafkaPushConf
	PushCron          string
	PushCronPoint     string
	TerminalBind      map[string]string
	TerminalList      []string
}
