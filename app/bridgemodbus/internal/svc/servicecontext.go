package svc

import (
	"context"
	"errors"
	"zero-service/app/bridgemodbus/internal/config"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/common/tool"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

type ServiceContext struct {
	Config                config.Config
	DB                    *gormx.DB
	ModbusConfigConverter *gormmodel.ModbusConfigConverter
	ModbusClientPool      *modbusx.ModbusClientPool
	Manager               *modbusx.PoolManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	// 创建 gormx 数据库连接
	db := gormx.MustOpenWithConf(c.DB)

	if isDevOrTest(c.Mode) {
		db.MustAutoMigrate(&gormmodel.ModbusSlaveConfig{})
	}

	return &ServiceContext{
		Config:                c,
		DB:                    db,
		ModbusConfigConverter: gormmodel.NewModbusConfigConverter(),
		ModbusClientPool:      modbusx.NewModbusClientPool(&c.ModbusClientConf, c.ModbusPool),
		Manager:               modbusx.NewPoolManager(),
	}
}

// isDevOrTest 判断是否为开发或测试环境
func isDevOrTest(mode string) bool {
	return mode == service.DevMode || mode == service.TestMode
}

func (s *ServiceContext) AddPool(ctx context.Context, modbusCode string) (*modbusx.ModbusClientPool, error) {
	if modbusCode == "" {
		return nil, errors.New("modbusCode不能为空")
	}
	var slaveConfig gormmodel.ModbusSlaveConfig
	err := s.DB.WithContext(ctx).Where("modbus_code = ?", modbusCode).First(&slaveConfig).Error
	if err != nil {
		return nil, err
	}
	if slaveConfig.Status != 1 {
		return nil, errors.New("配置不存在或未启用: " + modbusCode)
	}
	clientConf := s.ModbusConfigConverter.ToClientConf(&slaveConfig)
	if clientConf == nil {
		return nil, errors.New("配置转换失败")
	}
	pool, err := s.Manager.AddPool(ctx, modbusCode, clientConf, s.Config.ModbusPool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// GetModbusClientPool 获取 Modbus 连接池
// 如果 modbusCode 为空，返回默认连接池
// 如果 modbusCode 不为空，尝试从管理器获取连接池，如果不存在则创建新的连接池
func (s *ServiceContext) GetModbusClientPool(ctx context.Context, modbusCode string) (*modbusx.ModbusClientPool, error) {
	if len(modbusCode) == 0 {
		return s.ModbusClientPool, nil
	}

	var mdCliPool *modbusx.ModbusClientPool
	var ok bool
	mdCliPool, ok = s.Manager.GetPool(modbusCode)
	if !ok {
		var err error
		mdCliPool, err = s.AddPool(ctx, modbusCode)
		if err != nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "创建Modbus连接池失败")
		}
	}

	if mdCliPool == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "获取的Modbus连接池为空")
	}

	return mdCliPool, nil
}
