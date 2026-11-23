package bytex

import (
	"errors"
	"fmt"
	"math"
)

// ErrInvalidWordOrder 表示字序取值不在支持的四种排列内。
var ErrInvalidWordOrder = errors.New("bytex: 非法字序")

// WordOrder 描述 32 位值拆分到两个 16 位寄存器时，设备使用的字序与字节序。
//
// 以 0x12345678 为例（寄存器按设备返回顺序排列）：
//
//	ABCD: 0x1234 0x5678  高字在前，字内高字节在前（最常见）
//	CDAB: 0x5678 0x1234  低字在前（字交换）
//	BADC: 0x3412 0x7856  字内低字节在前（字节交换）
//	DCBA: 0x7856 0x3412  低字在前且字内低字节在前（整体小端）
//
// 四种排列的来源：Modbus 只定义 16 位寄存器，32 位值由厂商自行拆成两个寄存器，
// 规范未规定顺序；固件受 CPU 大小端和变量摆放影响，字顺序（高/低字在前）与
// 字内字节序（高/低字节在前）互相独立，2×2 即得四种。线上每寄存器固定高字节
// 先发，所以 BADC/DCBA 是固件把小端存放的字节按大端发出后的结果。
// 具体采用哪种只能以设备手册或实测结果为准，本包不做猜测：
// 零值是 WordOrderUnspecified，所有转换函数都会拒绝。
type WordOrder uint8

const (
	WordOrderUnspecified WordOrder = iota // 未指定，转换函数返回 ErrInvalidWordOrder
	WordOrderABCD
	WordOrderCDAB
	WordOrderBADC
	WordOrderDCBA
)

// Valid 判断字序是否为支持的四种排列。
func (o WordOrder) Valid() bool {
	switch o {
	case WordOrderABCD, WordOrderCDAB, WordOrderBADC, WordOrderDCBA:
		return true
	default:
		return false
	}
}

// String 返回字序名称，未指定返回 "unspecified"，其他非法值返回 "invalid"。
func (o WordOrder) String() string {
	switch o {
	case WordOrderUnspecified:
		return "unspecified"
	case WordOrderABCD:
		return "ABCD"
	case WordOrderCDAB:
		return "CDAB"
	case WordOrderBADC:
		return "BADC"
	case WordOrderDCBA:
		return "DCBA"
	default:
		return "invalid"
	}
}

// canonicalBytes 将设备顺序的两个寄存器还原为 ABCD 顺序的 4 字节。
func canonicalBytes(regs [2]uint16, order WordOrder) ([4]byte, error) {
	if !order.Valid() {
		return [4]byte{}, fmt.Errorf("%w: %d", ErrInvalidWordOrder, order)
	}
	r0h, r0l := byte(regs[0]>>8), byte(regs[0])
	r1h, r1l := byte(regs[1]>>8), byte(regs[1])

	switch order {
	case WordOrderABCD:
		return [4]byte{r0h, r0l, r1h, r1l}, nil
	case WordOrderCDAB:
		return [4]byte{r1h, r1l, r0h, r0l}, nil
	case WordOrderBADC:
		return [4]byte{r0l, r0h, r1l, r1h}, nil
	default: // WordOrderDCBA
		return [4]byte{r1l, r1h, r0l, r0h}, nil
	}
}

// bytesToRegisters 将 ABCD 顺序的 4 字节按字序拆成设备顺序的两个寄存器。
func bytesToRegisters(b [4]byte, order WordOrder) ([2]uint16, error) {
	if !order.Valid() {
		return [2]uint16{}, fmt.Errorf("%w: %d", ErrInvalidWordOrder, order)
	}
	hi := uint16(b[0])<<8 | uint16(b[1])
	lo := uint16(b[2])<<8 | uint16(b[3])
	swap := func(v uint16) uint16 { return v<<8 | v>>8 }

	switch order {
	case WordOrderABCD:
		return [2]uint16{hi, lo}, nil
	case WordOrderCDAB:
		return [2]uint16{lo, hi}, nil
	case WordOrderBADC:
		return [2]uint16{swap(hi), swap(lo)}, nil
	default: // WordOrderDCBA
		return [2]uint16{swap(lo), swap(hi)}, nil
	}
}

// CombineUint32 把设备顺序的两个寄存器还原为 uint32。
func CombineUint32(regs [2]uint16, order WordOrder) (uint32, error) {
	b, err := canonicalBytes(regs, order)
	if err != nil {
		return 0, err
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
}

// SplitUint32 把 uint32 按字序拆成设备顺序的两个寄存器。
func SplitUint32(v uint32, order WordOrder) ([2]uint16, error) {
	return bytesToRegisters([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}, order)
}

// CombineInt32 把设备顺序的两个寄存器还原为 int32（补码）。
func CombineInt32(regs [2]uint16, order WordOrder) (int32, error) {
	v, err := CombineUint32(regs, order)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// SplitInt32 把 int32 按字序拆成设备顺序的两个寄存器。
func SplitInt32(v int32, order WordOrder) ([2]uint16, error) {
	return SplitUint32(uint32(v), order)
}

// CombineFloat32 把设备顺序的两个寄存器还原为 float32（IEEE 754）。
func CombineFloat32(regs [2]uint16, order WordOrder) (float32, error) {
	v, err := CombineUint32(regs, order)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

// SplitFloat32 把 float32 按字序拆成设备顺序的两个寄存器。
func SplitFloat32(v float32, order WordOrder) ([2]uint16, error) {
	return SplitUint32(math.Float32bits(v), order)
}
