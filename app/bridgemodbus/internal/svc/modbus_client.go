package svc

import (
	"context"
	"errors"
	"zero-service/common/errx"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"gorm.io/gorm"
)

// GetModbusClient 复用每个 code 的长期客户端，不需要借还。
// 空 code 使用文件默认配置，非空 code 必须查询启用的表配置，不回退文件配置。
// 配置加载错误原样返回（已带平台码）；建连失败和管理器内部错误统一收敛为 100140。
func (s *ServiceContext) GetModbusClient(ctx context.Context, modbusCode string) (*modbusx.ModbusClient, error) {
	load := func(ctx context.Context) (*modbusx.ModbusClientConf, error) {
		if modbusCode == "" {
			return &s.Config.ModbusClientConf, nil
		}
		return s.loadModbusClientConfig(ctx, modbusCode)
	}

	cli, err := s.Manager.GetClient(ctx, modbusCode, load)
	if errors.Is(err, modbusx.ErrConfigChanged) {
		// 加载发布后被并发移除，尚未发生设备 I/O，重取一次即可。
		cli, err = s.Manager.GetClient(ctx, modbusCode, load)
	}
	if err == nil {
		return cli, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	// loadModbusClientConfig 的错误已带平台码：不存在/未启用/转换失败包装了
	// ErrConfigUnavailable，查库失败是 100050；NewErrorByPbCodeWrap 会保留原码和文案。

	message := "获取 Modbus 客户端失败"
	switch {
	case errors.Is(err, modbusx.ErrManagerClosed):
		message = "服务正在关闭，请稍后重试"
	case errors.Is(err, modbusx.ErrConfigChanged):
		message = "Modbus 配置已变更，请重新发起请求"
	}
	return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ, err, message)
}

// loadModbusClientConfig 查询当前已提交的表配置，错误保持原平台码返回。
func (s *ServiceContext) loadModbusClientConfig(ctx context.Context, modbusCode string) (*modbusx.ModbusClientConf, error) {
	var slaveConfig gormmodel.ModbusSlaveConfig
	// 建连与后台周期校验都会查询配置，成功查询不逐条打印 SQL；错误和慢查询仍会记录。
	err := s.DB.WithContext(gormx.WithoutSQLTrace(ctx)).
		Where("modbus_code = ?", modbusCode).
		First(&slaveConfig).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_RECORD_NOT_EXIST,
			modbusx.ErrConfigUnavailable, "Modbus配置不存在: "+modbusCode)
	}
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询Modbus配置失败")
	}
	if slaveConfig.Status != 1 {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ,
			modbusx.ErrConfigUnavailable, "配置未启用: "+modbusCode)
	}
	// 历史行可能因旧版接口省略 slave 而落库为 0；不能静默套用文件默认的 1，
	// 否则请求会指向同一网关下的另一台设备。非法值按不可用处理，后台校验会移除缓存。
	if slaveConfig.Slave < modbusx.MinSlave || slaveConfig.Slave > modbusx.MaxSlave {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ,
			modbusx.ErrConfigUnavailable,
			"配置从站地址非法: %s（slave=%d，要求 [%d, %d]）",
			modbusCode, slaveConfig.Slave, modbusx.MinSlave, modbusx.MaxSlave)
	}
	clientConf := s.ModbusConfigConverter.ToClientConf(&slaveConfig)
	if clientConf == nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ,
			modbusx.ErrConfigUnavailable, "配置转换失败")
	}
	return clientConf, nil
}
