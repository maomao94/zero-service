package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/common/modbusx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/grid-x/modbus"
)

// Modbus 单次请求的数量上限，来自 Modbus Application Protocol V1.1b3。
const (
	maxReadBits                = 2000 // 0x01 / 0x02
	maxWriteBits               = 1968 // 0x0F
	maxReadRegisters           = 125  // 0x03 / 0x04
	maxWriteRegisters          = 123  // 0x10
	maxReadWriteWriteRegisters = 121  // 0x17 的写入段
)

// validateAddress 校验起始地址并转换为协议使用的 uint16。
// 不做校验直接转换会静默截断（如 70000 → 4464），把数据写到错误地址。
func validateAddress(field string, value uint32) (uint16, error) {
	if value > 0xFFFF {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"%s 超出 Modbus 地址范围 [0, 65535]（当前 %d）", field, value)
	}
	return uint16(value), nil
}

// validateQuantity 校验单次读写数量，超出功能码上限时拒绝请求。
func validateQuantity(field string, value uint32, max int) (uint16, error) {
	if value == 0 || value > uint32(max) {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"%s 超出范围 [1, %d]（当前 %d）", field, max, value)
	}
	return uint16(value), nil
}

// validateValuesCount 校验 quantity 与 values 长度一致，避免少写/多写寄存器。
func validateValuesCount(field string, quantity uint32, count int) error {
	if int(quantity) != count {
		return tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"%s 与 values 数量不一致（%s=%d，len(values)=%d）", field, field, quantity, count)
	}
	return nil
}

// validateReadDeviceIDCode 校验设备标识读取类型并转换为 grid-x 的枚举。
//
// 本入口只接受 1-3：0x01 基本、0x02 常规、0x03 扩展。
// 0x04（指定对象）必须走 ReadDeviceIdentificationSpecificObject 单独传 objectId，
// grid-x v1.5.1 的 ReadDeviceIdentification 对 0x04 会直接返回 unsupported，
// 因此这里不接受 0x04，避免调用方误以为该入口支持指定对象。
func validateReadDeviceIDCode(value uint32) (modbus.ReadDeviceIDCode, error) {
	if value < uint32(modbus.ReadDeviceIDCodeBasic) || value > uint32(modbus.ReadDeviceIDCodeExtended) {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"readDeviceIdCode 超出范围 [1, 3]（当前 %d）", value)
	}
	return modbus.ReadDeviceIDCode(value), nil
}

// validate16BitValue 校验寄存器值/掩码是否在 16 位范围内。
func validate16BitValue(field string, value uint32) (uint16, error) {
	if value > 0xFFFF {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"%s 超出 16 位寄存器范围 [0, 65535]（当前 %d）", field, value)
	}
	return uint16(value), nil
}

// wrapModbusError 把设备异常响应与通信故障转换成平台错误码。
// 设备异常码（厂商码）只放在 100603 的 msg 中，不映射成平台码。
func wrapModbusError(action string, err error) error {
	return wrapModbusOperationError(action, err, false)
}

// wrapModbusWriteError 保留错误码，但通信失败时不承诺写入未执行。
// 超时、断链和流错位都可能发生在设备执行之后，调用方应先核实状态。
func wrapModbusWriteError(action string, err error) error {
	return wrapModbusOperationError(action, err, true)
}

func wrapModbusOperationError(action string, err error, write bool) error {
	if err == nil {
		return nil
	}
	if functionCode, exceptionCode, ok := modbusx.Exception(err); ok {
		return tool.NewThirdPartyAPIError(
			"modbus",
			fmt.Sprintf("0x%02X", exceptionCode),
			fmt.Sprintf("%s失败：功能码 0x%02X 返回异常（%s）",
				action, functionCode&0x7F, modbusx.ExceptionMessage(exceptionCode)),
		)
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	// 客户端已被管理器作废：callClient 在进入 SDK 前返回，请求确定未下发。
	// 不能走 write 分支的"执行结果未知"，否则会误导调用方去核实设备。
	if errors.Is(err, modbusx.ErrClientInvalidated) {
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ, err,
			"%s未执行：Modbus 客户端已失效，请重新发起请求", action)
	}
	if write {
		code := extproto.Code__1_00_THIRD_PARTY
		if modbusx.IsTimeout(err) {
			code = extproto.Code__1_00_TIMEOUT
		}
		return tool.NewErrorByPbCodeWrap(code, err,
			"%s失败：写入执行结果未知，请核实设备状态后决定是否重试", action)
	}
	if modbusx.IsTimeout(err) {
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_TIMEOUT, err, "%s超时", action)
	}
	if modbusx.IsStreamDesynced(err) {
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_THIRD_PARTY, err,
			"%s失败：Modbus 响应流错位，请重新发起读取", action)
	}
	return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_THIRD_PARTY, err, "%s失败：Modbus 通信异常", action)
}
