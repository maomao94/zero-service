package svc

import (
	"zero-service/app/bridgemodbus/internal/config"
	"zero-service/common/modbusx"
)

type ServiceContext struct {
	Config           config.Config
	ModbusClientPool *modbusx.ModbusClientPool
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:           c,
		ModbusClientPool: modbusx.NewModbusClientPool(&c.ModbusClientConf, 3),
	}
}
