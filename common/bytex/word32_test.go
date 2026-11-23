package bytex

import (
	"errors"
	"math"
	"testing"
)

func TestCombineUint32WordOrder(t *testing.T) {
	const want uint32 = 0x12345678
	cases := []struct {
		name string
		regs [2]uint16
		ord  WordOrder
	}{
		{"ABCD", [2]uint16{0x1234, 0x5678}, WordOrderABCD},
		{"CDAB", [2]uint16{0x5678, 0x1234}, WordOrderCDAB},
		{"BADC", [2]uint16{0x3412, 0x7856}, WordOrderBADC},
		{"DCBA", [2]uint16{0x7856, 0x3412}, WordOrderDCBA},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CombineUint32(c.regs, c.ord)
			if err != nil {
				t.Fatalf("CombineUint32() error = %v", err)
			}
			if got != want {
				t.Fatalf("CombineUint32() = 0x%08X, want 0x%08X", got, want)
			}
		})
	}
}

func TestSplitUint32WordOrder(t *testing.T) {
	const v uint32 = 0x12345678
	cases := []struct {
		name string
		ord  WordOrder
		want [2]uint16
	}{
		{"ABCD", WordOrderABCD, [2]uint16{0x1234, 0x5678}},
		{"CDAB", WordOrderCDAB, [2]uint16{0x5678, 0x1234}},
		{"BADC", WordOrderBADC, [2]uint16{0x3412, 0x7856}},
		{"DCBA", WordOrderDCBA, [2]uint16{0x7856, 0x3412}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SplitUint32(v, c.ord)
			if err != nil {
				t.Fatalf("SplitUint32() error = %v", err)
			}
			if got != c.want {
				t.Fatalf("SplitUint32() = %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestUint32WordOrderRoundTrip(t *testing.T) {
	values := []uint32{0, 1, 0xFFFF, 0x00010000, 0x12345678, math.MaxUint32}
	orders := []WordOrder{WordOrderABCD, WordOrderCDAB, WordOrderBADC, WordOrderDCBA}
	for _, v := range values {
		for _, ord := range orders {
			regs, err := SplitUint32(v, ord)
			if err != nil {
				t.Fatalf("SplitUint32(0x%08X, %s) error = %v", v, ord, err)
			}
			got, err := CombineUint32(regs, ord)
			if err != nil {
				t.Fatalf("CombineUint32(%#v, %s) error = %v", regs, ord, err)
			}
			if got != v {
				t.Fatalf("round trip 0x%08X via %s = 0x%08X", v, ord, got)
			}
		}
	}
}

func TestInt32WordOrder(t *testing.T) {
	const want int32 = -12
	// -12 的 32 位补码为 0xFFFFFFF4
	cases := []struct {
		name string
		ord  WordOrder
		regs [2]uint16
	}{
		{"ABCD", WordOrderABCD, [2]uint16{0xFFFF, 0xFFF4}},
		{"CDAB", WordOrderCDAB, [2]uint16{0xFFF4, 0xFFFF}},
		{"BADC", WordOrderBADC, [2]uint16{0xFFFF, 0xF4FF}},
		{"DCBA", WordOrderDCBA, [2]uint16{0xF4FF, 0xFFFF}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CombineInt32(c.regs, c.ord)
			if err != nil {
				t.Fatalf("CombineInt32() error = %v", err)
			}
			if got != want {
				t.Fatalf("CombineInt32() = %d, want %d", got, want)
			}
			regs, err := SplitInt32(want, c.ord)
			if err != nil {
				t.Fatalf("SplitInt32() error = %v", err)
			}
			if regs != c.regs {
				t.Fatalf("SplitInt32() = %#v, want %#v", regs, c.regs)
			}
		})
	}
}

func TestFloat32WordOrder(t *testing.T) {
	const want float32 = 220.5
	// 220.5 的 IEEE 754 单精度位模式为 0x435C8000
	cases := []struct {
		name string
		ord  WordOrder
		regs [2]uint16
	}{
		{"ABCD", WordOrderABCD, [2]uint16{0x435C, 0x8000}},
		{"CDAB", WordOrderCDAB, [2]uint16{0x8000, 0x435C}},
		{"BADC", WordOrderBADC, [2]uint16{0x5C43, 0x0080}},
		{"DCBA", WordOrderDCBA, [2]uint16{0x0080, 0x5C43}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CombineFloat32(c.regs, c.ord)
			if err != nil {
				t.Fatalf("CombineFloat32() error = %v", err)
			}
			if got != want {
				t.Fatalf("CombineFloat32() = %v, want %v", got, want)
			}
			regs, err := SplitFloat32(want, c.ord)
			if err != nil {
				t.Fatalf("SplitFloat32() error = %v", err)
			}
			if regs != c.regs {
				t.Fatalf("SplitFloat32() = %#v, want %#v", regs, c.regs)
			}
		})
	}
}

func TestInvalidWordOrder(t *testing.T) {
	// 零值是未指定，必须被拒绝，避免忘记设置字序时静默按 ABCD 处理。
	zero := WordOrderUnspecified
	if zero.Valid() {
		t.Fatal("unspecified order should not be valid")
	}
	if got := zero.String(); got != "unspecified" {
		t.Fatalf("String() = %q, want unspecified", got)
	}
	if _, err := SplitUint32(1, zero); !errors.Is(err, ErrInvalidWordOrder) {
		t.Fatalf("SplitUint32() error = %v, want ErrInvalidWordOrder", err)
	}

	bad := WordOrder(99)
	if bad.Valid() {
		t.Fatal("invalid order should not be valid")
	}
	if got := bad.String(); got != "invalid" {
		t.Fatalf("String() = %q, want invalid", got)
	}
	if _, err := CombineUint32([2]uint16{1, 2}, bad); !errors.Is(err, ErrInvalidWordOrder) {
		t.Fatalf("CombineUint32() error = %v, want ErrInvalidWordOrder", err)
	}
	if _, err := SplitUint32(1, bad); !errors.Is(err, ErrInvalidWordOrder) {
		t.Fatalf("SplitUint32() error = %v, want ErrInvalidWordOrder", err)
	}
	if _, err := CombineFloat32([2]uint16{1, 2}, bad); !errors.Is(err, ErrInvalidWordOrder) {
		t.Fatalf("CombineFloat32() error = %v, want ErrInvalidWordOrder", err)
	}
}
