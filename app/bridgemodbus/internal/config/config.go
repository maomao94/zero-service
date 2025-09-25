package config

import (
	"zero-service/common/modbusx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ModbusClientConf modbusx.ModbusClientConf
}
