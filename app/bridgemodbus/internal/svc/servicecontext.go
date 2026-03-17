package svc

import (
	"context"
	"errors"
	"zero-service/app/bridgemodbus/internal/config"
	"zero-service/common/dbx"
	"zero-service/common/modbusx"
	"zero-service/model"
)

type ServiceContext struct {
	Config                 config.Config
	ModbusSlaveConfigModel model.ModbusSlaveConfigModel
	ModbusConfigConverter  *model.ModbusConfigConverter
	ModbusClientPool       *modbusx.ModbusClientPool
	Manager                *modbusx.PoolManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 解析数据库类型
	dbType := dbx.ParseDatabaseType(c.DB.DataSource)
	return &ServiceContext{
		Config:                 c,
		ModbusSlaveConfigModel: model.NewModbusSlaveConfigModelWithDBType(dbx.New(c.DB.DataSource), model.DatabaseType(dbType)),
		ModbusConfigConverter:  model.NewModbusConfigConverter(),
		ModbusClientPool:       modbusx.NewModbusClientPool(&c.ModbusClientConf, c.ModbusPool),
		Manager:                modbusx.NewPoolManager(),
	}
}

func (s *ServiceContext) AddPool(ctx context.Context, modbusCode string) (*modbusx.ModbusClientPool, error) {
	if modbusCode == "" {
		return nil, errors.New("modbusCode不能为空")
	}
	slaveConfig, err := s.ModbusSlaveConfigModel.FindOneByModbusCode(ctx, modbusCode)
	if err != nil {
		return nil, err
	}
	if slaveConfig == nil || slaveConfig.Status != 1 {
		return nil, errors.New("配置不存在或未启用: " + modbusCode)
	}
	clientConf := s.ModbusConfigConverter.ToClientConf(slaveConfig)
	if clientConf == nil {
		return nil, errors.New("配置转换失败")
	}
	pool, err := s.Manager.AddPool(ctx, modbusCode, clientConf, s.Config.ModbusPool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
