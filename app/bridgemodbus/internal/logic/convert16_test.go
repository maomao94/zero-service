package logic

import (
	"context"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

func TestBatchConvertValue16ToRegisters(t *testing.T) {
	cases := []struct {
		name       string
		values     []int32
		unsigned   bool
		wantUint16 []uint32
		wantInt16  []int32
		wantHex    []string
		wantBytes  []byte
	}{
		{
			name:       "无符号边界",
			values:     []int32{0, 1, 65535},
			unsigned:   true,
			wantUint16: []uint32{0, 1, 65535},
			wantInt16:  []int32{0, 1, -1},
			wantHex:    []string{"0x0000", "0x0001", "0xFFFF"},
			wantBytes:  []byte{0x00, 0x00, 0x00, 0x01, 0xFF, 0xFF},
		},
		{
			name:       "有符号补码",
			values:     []int32{-32768, -1, 32767},
			wantUint16: []uint32{32768, 65535, 32767},
			wantInt16:  []int32{-32768, -1, 32767},
			wantHex:    []string{"0x8000", "0xFFFF", "0x7FFF"},
			wantBytes:  []byte{0x80, 0x00, 0xFF, 0xFF, 0x7F, 0xFF},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertValue16ToRegistersLogic(context.Background(), nil)
			res, err := l.BatchConvertValue16ToRegisters(&bridgemodbus.BatchConvertValue16ToRegistersReq{
				Values:   c.values,
				Unsigned: c.unsigned,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res.Uint16Values) != len(c.wantUint16) {
				t.Fatalf("uint16Values = %#v, want %#v", res.Uint16Values, c.wantUint16)
			}
			for i, want := range c.wantUint16 {
				if res.Uint16Values[i] != want || res.Int16Values[i] != c.wantInt16[i] {
					t.Fatalf("values[%d] = uint16 %d / int16 %d, want %d / %d",
						i, res.Uint16Values[i], res.Int16Values[i], want, c.wantInt16[i])
				}
				if res.HexValues[i] != c.wantHex[i] {
					t.Fatalf("hexValues[%d] = %q, want %q", i, res.HexValues[i], c.wantHex[i])
				}
			}
			if string(res.Bytes) != string(c.wantBytes) {
				t.Fatalf("bytes = % X, want % X", res.Bytes, c.wantBytes)
			}
		})
	}
}

func TestBatchConvertValue16ToRegistersRejectsOutOfRange(t *testing.T) {
	cases := []struct {
		name     string
		values   []int32
		unsigned bool
	}{
		{"无符号负数", []int32{-1}, true},
		{"无符号超上限", []int32{65536}, true},
		{"有符号超上限", []int32{32768}, false},
		{"有符号超下限", []int32{-32769}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertValue16ToRegistersLogic(context.Background(), nil)
			_, err := l.BatchConvertValue16ToRegisters(&bridgemodbus.BatchConvertValue16ToRegistersReq{
				Values:   c.values,
				Unsigned: c.unsigned,
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
				t.Fatalf("code = %d, want %d: %v", code, extproto.Code__1_00_PARAM_INVALID, err)
			}
		})
	}
}
