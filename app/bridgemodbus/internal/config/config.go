package config

import (
	"zero-service/common/modbusx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ModbusPool  int `json:",default=32"`
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	DB struct {
		DataSource string
	}
	ModbusClientConf modbusx.ModbusClientConf
}
