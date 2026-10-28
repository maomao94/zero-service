package dispatch

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"

	"google.golang.org/protobuf/proto"
)

// newMockModbusRoutes 为 Modbus I/O 方法提供固定数据，便于在没有 Modbus 设备时联调控制台。
func newMockModbusRoutes() map[string]Handler {
	return map[string]Handler{
		MethodModbusReadCoils: mockModbus(
			func() *bridgemodbus.ReadCoilsReq { return &bridgemodbus.ReadCoilsReq{} },
			&bridgemodbus.ReadCoilsRes{Results: []byte{0x05}, Values: []bool{true, false, true, false}},
		),
		MethodModbusReadDiscreteInputs: mockModbus(
			func() *bridgemodbus.ReadDiscreteInputsReq { return &bridgemodbus.ReadDiscreteInputsReq{} },
			&bridgemodbus.ReadDiscreteInputsRes{Results: []byte{0x03}, Values: []bool{true, true, false, false}},
		),
		MethodModbusReadInputRegisters: mockModbus(
			func() *bridgemodbus.ReadInputRegistersReq { return &bridgemodbus.ReadInputRegistersReq{} },
			mockRegistersRes(2),
		),
		MethodModbusReadHoldingRegisters: mockModbus(
			func() *bridgemodbus.ReadHoldingRegistersReq { return &bridgemodbus.ReadHoldingRegistersReq{} },
			&bridgemodbus.ReadHoldingRegistersRes{
				Results:      []byte{0x00, 0x64, 0xFF, 0x9C},
				UintValues:   []uint32{100, 65436},
				IntValues:    []int32{100, -100},
				HexValues:    []string{"0x0064", "0xFF9C"},
				BinaryValues: []string{"0b0000000001100100", "0b1111111110011100"},
			},
		),
		MethodModbusWriteSingleCoil: mockModbus(
			func() *bridgemodbus.WriteSingleCoilReq { return &bridgemodbus.WriteSingleCoilReq{} },
			&bridgemodbus.WriteSingleCoilRes{Results: []byte{0xFF, 0x00}},
		),
		MethodModbusWriteMultipleCoils: mockModbus(
			func() *bridgemodbus.WriteMultipleCoilsReq { return &bridgemodbus.WriteMultipleCoilsReq{} },
			&bridgemodbus.WriteMultipleCoilsRes{Results: []byte{0x00, 0x04}},
		),
		MethodModbusWriteSingleRegister: mockModbus(
			func() *bridgemodbus.WriteSingleRegisterReq { return &bridgemodbus.WriteSingleRegisterReq{} },
			&bridgemodbus.WriteSingleRegisterRes{Results: []byte{0x00, 0x64}},
		),
		MethodModbusWriteSingleRegisterWithDecimal: mockModbus(
			func() *bridgemodbus.WriteSingleRegisterWithDecimalReq {
				return &bridgemodbus.WriteSingleRegisterWithDecimalReq{}
			},
			&bridgemodbus.WriteSingleRegisterWithDecimalRes{Results: []byte{0x00, 0x64}},
		),
		MethodModbusWriteMultipleRegisters: mockModbus(
			func() *bridgemodbus.WriteMultipleRegistersReq { return &bridgemodbus.WriteMultipleRegistersReq{} },
			&bridgemodbus.WriteMultipleRegistersRes{Results: []byte{0x00, 0x02}},
		),
		MethodModbusWriteMultipleRegistersWithDecimal: mockModbus(
			func() *bridgemodbus.WriteMultipleRegistersWithDecimalReq {
				return &bridgemodbus.WriteMultipleRegistersWithDecimalReq{}
			},
			&bridgemodbus.WriteMultipleRegistersWithDecimalRes{Results: []byte{0x00, 0x02}},
		),
		MethodModbusReadWriteMultipleRegisters: mockModbus(
			func() *bridgemodbus.ReadWriteMultipleRegistersReq {
				return &bridgemodbus.ReadWriteMultipleRegistersReq{}
			},
			&bridgemodbus.ReadWriteMultipleRegistersRes{
				Results:      []byte{0x00, 0x64, 0xFF, 0x9C},
				UintValues:   []uint32{100, 65436},
				IntValues:    []int32{100, -100},
				HexValues:    []string{"0x0064", "0xFF9C"},
				BinaryValues: []string{"0b0000000001100100", "0b1111111110011100"},
			},
		),
		MethodModbusMaskWriteRegister: mockModbus(
			func() *bridgemodbus.MaskWriteRegisterReq { return &bridgemodbus.MaskWriteRegisterReq{} },
			&bridgemodbus.MaskWriteRegisterRes{Results: []byte{0x00, 0x64}},
		),
		MethodModbusReadFIFOQueue: mockModbus(
			func() *bridgemodbus.ReadFIFOQueueReq { return &bridgemodbus.ReadFIFOQueueReq{} },
			&bridgemodbus.ReadFIFOQueueRes{Results: []byte{0x00, 0x02, 0x00, 0x64, 0x00, 0xC8}},
		),
		MethodModbusReadDeviceIdentification: mockModbus(
			func() *bridgemodbus.ReadDeviceIdentificationReq {
				return &bridgemodbus.ReadDeviceIdentificationReq{}
			},
			&bridgemodbus.ReadDeviceIdentificationRes{
				Results:         map[uint32]string{0: "Mock Vendor", 1: "Mock Product", 2: "V1.0.0"},
				HexResults:      map[string]string{"0x00": "Mock Vendor", "0x01": "Mock Product", "0x02": "V1.0.0"},
				SemanticResults: map[string]string{"VendorName": "Mock Vendor", "ProductCode": "Mock Product", "MajorMinorRevision": "V1.0.0"},
			},
		),
		MethodModbusReadDeviceIdentificationSpecificObject: mockModbus(
			func() *bridgemodbus.ReadDeviceIdentificationSpecificObjectReq {
				return &bridgemodbus.ReadDeviceIdentificationSpecificObjectReq{}
			},
			&bridgemodbus.ReadDeviceIdentificationSpecificObjectRes{
				Results:         map[uint32]string{0: "Mock Vendor"},
				HexResults:      map[string]string{"0x00": "Mock Vendor"},
				SemanticResults: map[string]string{"VendorName": "Mock Vendor"},
			},
		),
	}
}

func mockRegistersRes(count int) *bridgemodbus.ReadInputRegistersRes {
	values := make([]uint32, 0, count)
	intValues := make([]int32, 0, count)
	hexValues := make([]string, 0, count)
	binaryValues := make([]string, 0, count)
	for i := 0; i < count; i++ {
		value := uint32(100 + i)
		values = append(values, value)
		intValues = append(intValues, int32(value))
		hexValues = append(hexValues, "0x0064")
		binaryValues = append(binaryValues, "0b0000000001100100")
	}
	return &bridgemodbus.ReadInputRegistersRes{
		Results:      []byte{0x00, 0x64, 0x00, 0x65},
		UintValues:   values,
		IntValues:    intValues,
		HexValues:    hexValues,
		BinaryValues: binaryValues,
	}
}

func mockModbus[T proto.Message](reqFactory func() T, response proto.Message) Handler {
	return func(_ context.Context, data string) (proto.Message, error) {
		req := reqFactory()
		if err := decode(data, req); err != nil {
			return nil, err
		}
		return response, nil
	}
}
