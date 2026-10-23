package dispatch

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"

	"google.golang.org/protobuf/proto"
)

// Modbus 开放 method。只开放 I/O 能力，配置管理和本地换算工具暂不对外。
const (
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

func newModbusRoutes(client bridgemodbus.BridgeModbusClient) map[string]Handler {
	return map[string]Handler{
		MethodModbusReadCoils: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadCoilsReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadCoils(ctx, &in)
		},
		MethodModbusReadDiscreteInputs: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadDiscreteInputsReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadDiscreteInputs(ctx, &in)
		},
		MethodModbusReadInputRegisters: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadInputRegistersReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadInputRegisters(ctx, &in)
		},
		MethodModbusReadHoldingRegisters: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadHoldingRegistersReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadHoldingRegisters(ctx, &in)
		},
		MethodModbusWriteSingleCoil: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteSingleCoilReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteSingleCoil(ctx, &in)
		},
		MethodModbusWriteMultipleCoils: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteMultipleCoilsReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteMultipleCoils(ctx, &in)
		},
		MethodModbusWriteSingleRegister: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteSingleRegisterReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteSingleRegister(ctx, &in)
		},
		MethodModbusWriteSingleRegisterWithDecimal: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteSingleRegisterWithDecimalReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteSingleRegisterWithDecimal(ctx, &in)
		},
		MethodModbusWriteMultipleRegisters: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteMultipleRegistersReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteMultipleRegisters(ctx, &in)
		},
		MethodModbusWriteMultipleRegistersWithDecimal: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.WriteMultipleRegistersWithDecimalReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.WriteMultipleRegistersWithDecimal(ctx, &in)
		},
		MethodModbusReadWriteMultipleRegisters: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadWriteMultipleRegistersReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadWriteMultipleRegisters(ctx, &in)
		},
		MethodModbusMaskWriteRegister: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.MaskWriteRegisterReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.MaskWriteRegister(ctx, &in)
		},
		MethodModbusReadFIFOQueue: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadFIFOQueueReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadFIFOQueue(ctx, &in)
		},
		MethodModbusReadDeviceIdentification: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadDeviceIdentificationReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadDeviceIdentification(ctx, &in)
		},
		MethodModbusReadDeviceIdentificationSpecificObject: func(ctx context.Context, data []byte) (proto.Message, error) {
			var in bridgemodbus.ReadDeviceIdentificationSpecificObjectReq
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return client.ReadDeviceIdentificationSpecificObject(ctx, &in)
		},
	}
}
