package bytex

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

// refRegisters 用标准库字节解码器独立推导设备寄存器序列，作为字序实现的对照实现：
// 每个寄存器就是"同一段四字节按大端或小端读出的 16 位值"，再由字序决定两个寄存器的先后。
// 这样期望值不依赖本包的移位与交换写法，能发现编码与解码同时按同一错误定义实现的镜像问题。
func refRegisters(b [4]byte, order WordOrder) [2]uint16 {
	be0 := binary.BigEndian.Uint16(b[0:2])
	be1 := binary.BigEndian.Uint16(b[2:4])
	le0 := binary.LittleEndian.Uint16(b[0:2])
	le1 := binary.LittleEndian.Uint16(b[2:4])
	switch order {
	case WordOrderABCD: // 高字在前、字内高字节在前
		return [2]uint16{be0, be1}
	case WordOrderCDAB: // 字交换
		return [2]uint16{be1, be0}
	case WordOrderBADC: // 字内字节交换
		return [2]uint16{le0, le1}
	default: // WordOrderDCBA：整体小端
		return [2]uint16{le1, le0}
	}
}

// TestWordOrderAgainstEncodingBinary 用随机值与关键锚点对拍四种字序的拆分与还原。
func TestWordOrderAgainstEncodingBinary(t *testing.T) {
	rng := rand.New(rand.NewSource(20261123))
	words := make([][4]byte, 0, 2000)
	for range 2000 {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], rng.Uint32())
		words = append(words, b)
	}
	words = append(words,
		[4]byte{0x00, 0x00, 0x00, 0x00},
		[4]byte{0x00, 0x00, 0x00, 0x01},
		[4]byte{0x12, 0x34, 0x56, 0x78},
		[4]byte{0x80, 0x00, 0x00, 0x00},
		[4]byte{0x00, 0x00, 0x80, 0x00},
		[4]byte{0xFF, 0xFF, 0xFF, 0xFF},
	)

	orders := []WordOrder{WordOrderABCD, WordOrderCDAB, WordOrderBADC, WordOrderDCBA}
	for _, b := range words {
		bits := binary.BigEndian.Uint32(b[:])
		for _, order := range orders {
			want := refRegisters(b, order)

			gotU, err := SplitUint32(bits, order)
			if err != nil {
				t.Fatalf("SplitUint32(0x%08X, %s) error = %v", bits, order, err)
			}
			if gotU != want {
				t.Fatalf("SplitUint32(0x%08X, %s) = %#v, want %#v", bits, order, gotU, want)
			}
			backU, err := CombineUint32(want, order)
			if err != nil {
				t.Fatalf("CombineUint32(%#v, %s) error = %v", want, order, err)
			}
			if backU != bits {
				t.Fatalf("CombineUint32(%#v, %s) = 0x%08X, want 0x%08X", want, order, backU, bits)
			}

			gotI, err := SplitInt32(int32(bits), order)
			if err != nil {
				t.Fatalf("SplitInt32(%d, %s) error = %v", int32(bits), order, err)
			}
			if gotI != want {
				t.Fatalf("SplitInt32(%d, %s) = %#v, want %#v", int32(bits), order, gotI, want)
			}
			backI, err := CombineInt32(want, order)
			if err != nil {
				t.Fatalf("CombineInt32(%#v, %s) error = %v", want, order, err)
			}
			if backI != int32(bits) {
				t.Fatalf("CombineInt32(%#v, %s) = %d, want %d", want, order, backI, int32(bits))
			}

			// float32 按位对拍，NaN/Inf 位模式也必须原样保留。
			f := math.Float32frombits(bits)
			gotF, err := SplitFloat32(f, order)
			if err != nil {
				t.Fatalf("SplitFloat32(%v, %s) error = %v", f, order, err)
			}
			if gotF != want {
				t.Fatalf("SplitFloat32(0x%08X, %s) = %#v, want %#v", bits, order, gotF, want)
			}
			backF, err := CombineFloat32(want, order)
			if err != nil {
				t.Fatalf("CombineFloat32(%#v, %s) error = %v", want, order, err)
			}
			if math.Float32bits(backF) != bits {
				t.Fatalf("CombineFloat32(%#v, %s) bits = 0x%08X, want 0x%08X", want, order, math.Float32bits(backF), bits)
			}
		}
	}
}
