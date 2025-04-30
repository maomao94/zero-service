package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/config"
	"zero-service/iec104/iec104client"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaASDUConfig config.KqConfig
	IecServerConfig iec104client.IecServerConfig
	CoaConfig       []iec104client.CoaConfig
}
