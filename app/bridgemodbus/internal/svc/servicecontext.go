package svc

import (
	"zero-service/app/bridgemodbus/internal/config"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

// ServiceContext 持有 bridgemodbus 的共享依赖。
type ServiceContext struct {
	Config                config.Config                    // 服务配置：数据库与文件默认客户端
	DB                    *gormx.DB                        // 数据库连接，用于按 modbusCode 查询从站配置
	ModbusConfigConverter *gormmodel.ModbusConfigConverter // 数据库模型到 Modbus 客户端配置的转换器
	Manager               *modbusx.ClientManager           // 空 code 使用文件配置，非空 code 使用表配置
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
		Manager:               modbusx.NewClientManager(),
	}
}

// isDevOrTest 判断是否为开发或测试环境
func isDevOrTest(mode string) bool {
	return mode == service.DevMode || mode == service.TestMode
}
