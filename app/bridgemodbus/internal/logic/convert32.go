package logic

import (
	"fmt"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/bytex"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

// maxRegisterValue 是 16 位寄存器的最大值。
const maxRegisterValue = 65535

// registersToUint32 把两个寄存器转换为响应使用的 uint32 列表（按设备顺序）。
func registersToUint32(regs [2]uint16) []uint32 {
	return bytex.Uint16SliceToUint32Slice(regs[:])
}

// registersToBytes 把两个寄存器转换为四个字节（按设备顺序，每寄存器 2 字节，高字节在前）。
func registersToBytes(regs [2]uint16) []byte {
	return bytex.Uint16SliceToBytes(regs[:])
}

// registersToHex 把两个寄存器格式化为设备顺序的十六进制文本，编码与解码共用，保证两侧格式一致。
func registersToHex(regs [2]uint16) string {
	return fmt.Sprintf("0x%04X 0x%04X", regs[0], regs[1])
}

// toWordOrder 把 proto 字序转换为 bytex 字序，未指定或未知值直接拒绝。
func toWordOrder(v bridgemodbus.WordOrder) (bytex.WordOrder, error) {
	switch v {
	case bridgemodbus.WordOrder_WORD_ORDER_ABCD:
		return bytex.WordOrderABCD, nil
	case bridgemodbus.WordOrder_WORD_ORDER_CDAB:
		return bytex.WordOrderCDAB, nil
	case bridgemodbus.WordOrder_WORD_ORDER_BADC:
		return bytex.WordOrderBADC, nil
	case bridgemodbus.WordOrder_WORD_ORDER_DCBA:
		return bytex.WordOrderDCBA, nil
	default:
		return 0, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"wordOrder 必须显式指定为 ABCD、CDAB、BADC 或 DCBA")
	}
}

// validateValue32Type 校验 32 位数值类型，未指定或未知值直接拒绝。
func validateValue32Type(t bridgemodbus.Value32Type) error {
	switch t {
	case bridgemodbus.Value32Type_VALUE32_UINT32,
		bridgemodbus.Value32Type_VALUE32_INT32,
		bridgemodbus.Value32Type_VALUE32_FLOAT32:
		return nil
	default:
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"dataType 必须显式指定为 uint32、int32 或 float32")
	}
}
