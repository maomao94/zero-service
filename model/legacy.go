package model

import "zero-service/model/gormmodel"

type ModbusSlaveConfig = gormmodel.ModbusSlaveConfig
type ModbusConfigConverter = gormmodel.ModbusConfigConverter

func NewModbusConfigConverter() *ModbusConfigConverter {
	return gormmodel.NewModbusConfigConverter()
}
