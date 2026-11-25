package dispatch

import "zero-service/app/bridgemodbus/bridgemodbus"

// Modbus 网关方法。
const (
	MethodModbusSaveConfig                             = "modbus.saveConfig"
	MethodModbusDeleteConfig                           = "modbus.deleteConfig"
	MethodModbusPageListConfig                         = "modbus.pageListConfig"
	MethodModbusGetConfigByCode                        = "modbus.getConfigByCode"
	MethodModbusBatchGetConfigByCode                   = "modbus.batchGetConfigByCode"
	MethodModbusBatchConvertValue16ToRegisters         = "modbus.batchConvertValue16ToRegisters"
	MethodModbusBatchConvertValue32ToRegisters         = "modbus.batchConvertValue32ToRegisters"
	MethodModbusBatchConvertRegistersToValue32         = "modbus.batchConvertRegistersToValue32"
	MethodModbusReadCoils                              = "modbus.readCoils"
	MethodModbusReadDiscreteInputs                     = "modbus.readDiscreteInputs"
	MethodModbusReadInputRegisters                     = "modbus.readInputRegisters"
	MethodModbusReadHoldingRegisters                   = "modbus.readHoldingRegisters"
	MethodModbusWriteSingleCoil                        = "modbus.writeSingleCoil"
	MethodModbusWriteMultipleCoils                     = "modbus.writeMultipleCoils"
	MethodModbusWriteSingleRegister                    = "modbus.writeSingleRegister"
	MethodModbusWriteSingleRegisterWithDecimal         = "modbus.writeSingleRegisterWithDecimal"
	MethodModbusWriteMultipleRegisters                 = "modbus.writeMultipleRegisters"
	MethodModbusWriteMultipleRegistersWithDecimal      = "modbus.writeMultipleRegistersWithDecimal"
	MethodModbusReadWriteMultipleRegisters             = "modbus.readWriteMultipleRegisters"
	MethodModbusMaskWriteRegister                      = "modbus.maskWriteRegister"
	MethodModbusReadFIFOQueue                          = "modbus.readFIFOQueue"
	MethodModbusReadDeviceIdentification               = "modbus.readDeviceIdentification"
	MethodModbusReadDeviceIdentificationSpecificObject = "modbus.readDeviceIdentificationSpecificObject"
)

func modbusRoutes(client bridgemodbus.BridgeModbusClient) map[string]Handler {
	return map[string]Handler{
		MethodModbusSaveConfig:                             relay(client.SaveConfig),
		MethodModbusDeleteConfig:                           relay(client.DeleteConfig),
		MethodModbusPageListConfig:                         relay(client.PageListConfig),
		MethodModbusGetConfigByCode:                        relay(client.GetConfigByCode),
		MethodModbusBatchGetConfigByCode:                   relay(client.BatchGetConfigByCode),
		MethodModbusBatchConvertValue16ToRegisters:         relay(client.BatchConvertValue16ToRegisters),
		MethodModbusBatchConvertValue32ToRegisters:         relay(client.BatchConvertValue32ToRegisters),
		MethodModbusBatchConvertRegistersToValue32:         relay(client.BatchConvertRegistersToValue32),
		MethodModbusReadCoils:                              relay(client.ReadCoils),
		MethodModbusReadDiscreteInputs:                     relay(client.ReadDiscreteInputs),
		MethodModbusReadInputRegisters:                     relay(client.ReadInputRegisters),
		MethodModbusReadHoldingRegisters:                   relay(client.ReadHoldingRegisters),
		MethodModbusWriteSingleCoil:                        relay(client.WriteSingleCoil),
		MethodModbusWriteMultipleCoils:                     relay(client.WriteMultipleCoils),
		MethodModbusWriteSingleRegister:                    relay(client.WriteSingleRegister),
		MethodModbusWriteSingleRegisterWithDecimal:         relay(client.WriteSingleRegisterWithDecimal),
		MethodModbusWriteMultipleRegisters:                 relay(client.WriteMultipleRegisters),
		MethodModbusWriteMultipleRegistersWithDecimal:      relay(client.WriteMultipleRegistersWithDecimal),
		MethodModbusReadWriteMultipleRegisters:             relay(client.ReadWriteMultipleRegisters),
		MethodModbusMaskWriteRegister:                      relay(client.MaskWriteRegister),
		MethodModbusReadFIFOQueue:                          relay(client.ReadFIFOQueue),
		MethodModbusReadDeviceIdentification:               relay(client.ReadDeviceIdentification),
		MethodModbusReadDeviceIdentificationSpecificObject: relay(client.ReadDeviceIdentificationSpecificObject),
	}
}
