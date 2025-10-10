package model

import "zero-service/common/modbusx"

// ModbusConfigConverter 用于数据库模型与客户端配置的转换
type ModbusConfigConverter struct{}

// NewModbusConfigConverter 创建转换工具实例
func NewModbusConfigConverter() *ModbusConfigConverter {
	return &ModbusConfigConverter{}
}

// ToClientConf 将数据库模型转换为Modbus客户端配置
// 入参：数据库查询得到的Modbus从站配置模型
// 出参：可直接用于初始化连接池的客户端配置
func (c *ModbusConfigConverter) ToClientConf(model *ModbusSlaveConfig) *modbusx.ModbusClientConf {
	if model == nil {
		return nil
	}

	// 基础连接配置
	conf := &modbusx.ModbusClientConf{
		Address:                 model.SlaveAddress,
		Slave:                   model.Slave,
		Timeout:                 int64(model.Timeout),
		IdleTimeout:             int64(model.IdleTimeout),
		LinkRecoveryTimeout:     int64(model.LinkRecoveryTimeout),
		ProtocolRecoveryTimeout: int64(model.ProtocolRecoveryTimeout),
		ConnectDelay:            int64(model.ConnectDelay),
	}

	// TLS配置（根据数据库字段判断是否启用）
	conf.TLS.Enable = model.EnableTls == 1
	if conf.TLS.Enable {
		conf.TLS.CertFile = model.TlsCertFile
		conf.TLS.KeyFile = model.TlsKeyFile
		conf.TLS.CAFile = model.TlsCaFile
	}

	return conf
}

// BatchToClientConf 批量转换数据库模型为客户端配置
// 入参：数据库查询得到的Modbus从站配置模型列表
// 出参：以modbus_code为key的客户端配置映射表
func (c *ModbusConfigConverter) BatchToClientConf(models []*ModbusSlaveConfig) map[string]*modbusx.ModbusClientConf {
	confMap := make(map[string]*modbusx.ModbusClientConf, len(models))
	for _, model := range models {
		if model == nil || model.ModbusCode == "" {
			continue // 跳过无效模型
		}
		confMap[model.ModbusCode] = c.ToClientConf(model)
	}
	return confMap
}
