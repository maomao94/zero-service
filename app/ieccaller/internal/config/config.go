package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/config"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaASDUConfig config.KqConfig

	Remote struct {
		Host        string
		Port        int
		Name        string
		DefaultName string
	}
}
