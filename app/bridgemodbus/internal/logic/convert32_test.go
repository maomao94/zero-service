package logic

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestBatchConvertValue32ToRegisters(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		dataType bridgemodbus.Value32Type
		order    bridgemodbus.WordOrder
		wantRegs []uint32
		wantHex  string
		wantBin  string
	}{
		{
			name:     "float32 CDAB",
			value:    220.5,
			dataType: bridgemodbus.Value32Type_VALUE32_FLOAT32,
			order:    bridgemodbus.WordOrder_WORD_ORDER_CDAB,
			wantRegs: []uint32{0x8000, 0x435C},
			wantHex:  "0x8000 0x435C",
			wantBin:  "1000000000000000 0100001101011100",
		},
		{
			name:     "int32 ABCD 负数",
			value:    -12,
			dataType: bridgemodbus.Value32Type_VALUE32_INT32,
			order:    bridgemodbus.WordOrder_WORD_ORDER_ABCD,
			wantRegs: []uint32{0xFFFF, 0xFFF4},
			wantHex:  "0xFFFF 0xFFF4",
		},
		{
			name:     "uint32 BADC",
			value:    0x12345678,
			dataType: bridgemodbus.Value32Type_VALUE32_UINT32,
			order:    bridgemodbus.WordOrder_WORD_ORDER_BADC,
			wantRegs: []uint32{0x3412, 0x7856},
			wantHex:  "0x3412 0x7856",
		},
		{
			// 上界必须被接受，防止范围校验写成 > 与 >= 混用。
			name:     "uint32 上界",
			value:    4294967295,
			dataType: bridgemodbus.Value32Type_VALUE32_UINT32,
			order:    bridgemodbus.WordOrder_WORD_ORDER_ABCD,
			wantRegs: []uint32{0xFFFF, 0xFFFF},
			wantHex:  "0xFFFF 0xFFFF",
			wantBin:  "1111111111111111 1111111111111111",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
			res, err := l.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
				Values:    []float64{c.value},
				DataType:  c.dataType,
				WordOrder: c.order,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res.Results) != 1 {
				t.Fatalf("results length = %d, want 1", len(res.Results))
			}
			got := res.Results[0]
			if len(got.Registers) != 2 || got.Registers[0] != c.wantRegs[0] || got.Registers[1] != c.wantRegs[1] {
				t.Fatalf("registers = %#v, want %#v", got.Registers, c.wantRegs)
			}
			if got.Hex != c.wantHex {
				t.Fatalf("hex = %q, want %q", got.Hex, c.wantHex)
			}
			if c.wantBin != "" && got.Binary != c.wantBin {
				t.Fatalf("binary = %q, want %q", got.Binary, c.wantBin)
			}
			if got.Value != c.value {
				t.Fatalf("value = %v, want %v", got.Value, c.value)
			}
			wantBytes := []byte{byte(c.wantRegs[0] >> 8), byte(c.wantRegs[0]), byte(c.wantRegs[1] >> 8), byte(c.wantRegs[1])}
			if string(got.Bytes) != string(wantBytes) {
				t.Fatalf("bytes = % X, want % X", got.Bytes, wantBytes)
			}
		})
	}
}

func TestBatchConvertValue32ToRegistersRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		req  *bridgemodbus.BatchConvertValue32ToRegistersReq
	}{
		{"空列表", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"未指定类型", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{1},
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"未指定字序", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:   []float64{1},
			DataType: bridgemodbus.Value32Type_VALUE32_UINT32,
		}},
		{"未知字序", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{1},
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder(99),
		}},
		{"uint32 负数", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{-1},
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"uint32 超上限", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{4294967296},
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"uint32 非整数", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{1.5},
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"int32 超上限", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{2147483648},
			DataType:  bridgemodbus.Value32Type_VALUE32_INT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"float32 NaN", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{math.NaN()},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"float32 正无穷", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{math.Inf(1)},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"float32 负无穷", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{math.Inf(-1)},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"float32 超上限", &bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{math.MaxFloat64},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
			_, err := l.BatchConvertValue32ToRegisters(c.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
				t.Fatalf("code = %d, want %d: %v", code, extproto.Code__1_00_PARAM_INVALID, err)
			}
		})
	}
}

// float32 编码回显的应是实际写入的 float32 载荷，而不是 float64 入参；
// 否则上游拿回显值和设备读回值比对时会看到无法解释的精度差。
func TestBatchConvertValue32ToRegistersFloatEcho(t *testing.T) {
	enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
	encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
		Values:    []float64{0.1},
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := encRes.Results[0]
	if got.Value == 0.1 {
		t.Fatal("回显不应保留 float64 精度，应回落为 float32 载荷")
	}
	if got.Value != float64(float32(0.1)) {
		t.Fatalf("value = %v, want %v", got.Value, float64(float32(0.1)))
	}

	dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: encRes.Registers,
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decRes.Results[0].Value != got.Value {
		t.Fatalf("解码主结果 %v 应与编码回显 %v 一致", decRes.Results[0].Value, got.Value)
	}
}

func TestBatchConvertRegistersToValue32(t *testing.T) {
	cases := []struct {
		name      string
		regs      []uint32
		dataType  bridgemodbus.Value32Type
		order     bridgemodbus.WordOrder
		wantValue float64
		wantU32   uint32
		wantI32   int32
		wantF32   float32
		wantHex   string
	}{
		{
			name:      "float32 CDAB",
			regs:      []uint32{0x8000, 0x435C},
			dataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			order:     bridgemodbus.WordOrder_WORD_ORDER_CDAB,
			wantValue: 220.5,
			wantU32:   0x435C8000,
			wantI32:   0x435C8000,
			wantF32:   220.5,
			wantHex:   "0x8000 0x435C",
		},
		{
			name:      "int32 ABCD 负数",
			regs:      []uint32{0xFFFF, 0xFFF4},
			dataType:  bridgemodbus.Value32Type_VALUE32_INT32,
			order:     bridgemodbus.WordOrder_WORD_ORDER_ABCD,
			wantValue: -12,
			wantU32:   0xFFFFFFF4,
			wantI32:   -12,
			wantHex:   "0xFFFF 0xFFF4",
		},
		{
			name:      "uint32 DCBA",
			regs:      []uint32{0x7856, 0x3412},
			dataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			order:     bridgemodbus.WordOrder_WORD_ORDER_DCBA,
			wantValue: 0x12345678,
			wantU32:   0x12345678,
			wantI32:   0x12345678,
			wantHex:   "0x7856 0x3412",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
			res, err := l.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
				Registers: c.regs,
				DataType:  c.dataType,
				WordOrder: c.order,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res.Results) != 1 {
				t.Fatalf("results length = %d, want 1", len(res.Results))
			}
			got := res.Results[0]
			if got.Value != c.wantValue {
				t.Fatalf("value = %v, want %v", got.Value, c.wantValue)
			}
			if got.Uint32Value != c.wantU32 {
				t.Fatalf("uint32Value = %d, want %d", got.Uint32Value, c.wantU32)
			}
			if got.Int32Value != c.wantI32 {
				t.Fatalf("int32Value = %d, want %d", got.Int32Value, c.wantI32)
			}
			if c.dataType == bridgemodbus.Value32Type_VALUE32_FLOAT32 {
				if got.Float32Value != c.wantF32 {
					t.Fatalf("float32Value = %v, want %v", got.Float32Value, c.wantF32)
				}
			} else if got.Float32Value != 0 {
				t.Fatalf("float32Value 不应在 %v 解码时填充，got %v", c.dataType, got.Float32Value)
			}
			if got.Hex != c.wantHex {
				t.Fatalf("hex = %q, want %q (设备顺序的两个寄存器)", got.Hex, c.wantHex)
			}
			if len(got.Registers) != 2 || got.Registers[0] != c.regs[0] || got.Registers[1] != c.regs[1] {
				t.Fatalf("registers = %#v, want %#v", got.Registers, c.regs)
			}
			wantBytes := []byte{byte(c.regs[0] >> 8), byte(c.regs[0]), byte(c.regs[1] >> 8), byte(c.regs[1])}
			if string(got.Bytes) != string(wantBytes) {
				t.Fatalf("bytes = % X, want % X", got.Bytes, wantBytes)
			}
		})
	}
}

// 0.12 的 float32 位模式是 0x3DF5C28F，精确展开成 double 是 0.11999999731779099。
// float32Value 应保持 float32 的最短表示 0.12，主结果 value 是精确展开值。
func TestValue32DecodeJSONKeepsShortFloat(t *testing.T) {
	l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	res, err := l.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: []uint32{0xC28F, 0x3DF5},
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := protojson.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `"float32Value":0.12`) {
		t.Fatalf("float32Value should be short in JSON, got %s", text)
	}
	if strings.Contains(text, `"float32Value":0.1199`) {
		t.Fatalf("float32Value should not carry float64 precision, got %s", text)
	}
	if !strings.Contains(text, `"value":0.11999999731779099`) {
		t.Fatalf("value should be the exact double of the float32 payload, got %s", text)
	}
}

// 整数解码不应把同一段比特当作浮点输出，JSON 里不能出现 NaN/Infinity 字符串。
func TestValue32DecodeJSONHasNoNonFiniteFloat(t *testing.T) {
	l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	res, err := l.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: []uint32{0xFFFF, 0xFFFF, 0x7F80, 0x0000},
		DataType:  bridgemodbus.Value32Type_VALUE32_INT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := protojson.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	text := string(raw)
	for _, bad := range []string{"NaN", "Infinity"} {
		if strings.Contains(text, bad) {
			t.Fatalf("JSON 不应出现 %s，got %s", bad, text)
		}
	}
	if !strings.Contains(text, `"value":-1`) {
		t.Fatalf("int32 -1 的 value 应为 -1，got %s", text)
	}
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
}

// 设备返回的 NaN/Inf 位模式要原样呈现（protobuf JSON 用 "NaN"/"Infinity" 字符串），
// 这是转码工具的预期行为：不改写设备数据。用测试锁定，避免后续被误当作 bug 修掉。
func TestValue32DecodeNonFiniteFloatJSON(t *testing.T) {
	l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	res, err := l.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		// 0x7FC00000 = quiet NaN，0x7F800000 = +Inf
		Registers: []uint32{0x7FC0, 0x0000, 0x7F80, 0x0000},
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
	raw, err := protojson.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `"value":"NaN"`) {
		t.Fatalf("NaN 位模式应以 \"NaN\" 字符串输出，got %s", text)
	}
	if !strings.Contains(text, `"value":"Infinity"`) {
		t.Fatalf("+Inf 位模式应以 \"Infinity\" 字符串输出，got %s", text)
	}
}

func TestBatchConvertRegistersToValue32RejectsInvalid(t *testing.T) {
	base := func(regs []uint32) *bridgemodbus.BatchConvertRegistersToValue32Req {
		return &bridgemodbus.BatchConvertRegistersToValue32Req{
			Registers: regs,
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}
	}
	cases := []struct {
		name string
		req  *bridgemodbus.BatchConvertRegistersToValue32Req
	}{
		{"空列表", &bridgemodbus.BatchConvertRegistersToValue32Req{
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"寄存器数为奇数", base([]uint32{0x1234, 0x5678, 0x9ABC})},
		{"单个寄存器", base([]uint32{0x1234})},
		{"寄存器超范围", base([]uint32{0x1234, 65536})},
		{"未指定类型", &bridgemodbus.BatchConvertRegistersToValue32Req{
			Registers: []uint32{0x1234, 0x5678},
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}},
		{"未指定字序", &bridgemodbus.BatchConvertRegistersToValue32Req{
			Registers: []uint32{0x1234, 0x5678},
			DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
			_, err := l.BatchConvertRegistersToValue32(c.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
				t.Fatalf("code = %d, want %d: %v", code, extproto.Code__1_00_PARAM_INVALID, err)
			}
		})
	}
}

// 多个值编码后应按输入顺序展平成连续寄存器与连续字节，并可按同字序解码还原。
func TestConvert32MultiValueRoundTrip(t *testing.T) {
	enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
	encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
		Values:    []float64{220.5, 0.12, -12.75},
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB,
	})
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if len(encRes.Results) != 3 {
		t.Fatalf("encode results = %d, want 3", len(encRes.Results))
	}
	wantRegs := []uint32{
		0x8000, 0x435C, // 220.5 CDAB
		0xC28F, 0x3DF5, // 0.12 CDAB
		0x0000, 0xC14C, // -12.75 CDAB
	}
	if len(encRes.Registers) != len(wantRegs) {
		t.Fatalf("registers = %#v, want %#v", encRes.Registers, wantRegs)
	}
	for i := range wantRegs {
		if encRes.Registers[i] != wantRegs[i] {
			t.Fatalf("registers = %#v, want %#v", encRes.Registers, wantRegs)
		}
	}
	// 逐值结果与展平后的寄存器保持一致，字节按设备顺序大端排列。
	for i, want := range wantRegs {
		if encRes.Results[i/2].Registers[i%2] != want {
			t.Fatalf("results[%d].registers = %#v, want[%d] = %#x", i/2, encRes.Results[i/2].Registers, i, want)
		}
	}
	wantBytes := []byte{0x80, 0x00, 0x43, 0x5C}
	if string(encRes.Results[0].Bytes) != string(wantBytes) {
		t.Fatalf("results[0].bytes = % X, want % X", encRes.Results[0].Bytes, wantBytes)
	}

	dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: encRes.Registers,
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB,
	})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decRes.Results) != 3 {
		t.Fatalf("decode results = %d, want 3", len(decRes.Results))
	}
	wantValues := []float64{220.5, 0.11999999731779099, -12.75}
	for i, want := range wantValues {
		got := decRes.Results[i]
		if got.Value != want {
			t.Fatalf("results[%d].value = %v, want %v", i, got.Value, want)
		}
		// 解码结果与编码结果对称：回显该组的寄存器、字节与十六进制都是设备顺序。
		if len(got.Registers) != 2 || got.Registers[0] != wantRegs[i*2] || got.Registers[1] != wantRegs[i*2+1] {
			t.Fatalf("results[%d].registers = %#v, want %#v", i, got.Registers, wantRegs[i*2:i*2+2])
		}
		if got.Hex != encRes.Results[i].Hex {
			t.Fatalf("results[%d].hex = %q，应与编码侧 %q 一致（设备顺序）", i, got.Hex, encRes.Results[i].Hex)
		}
		wantGroupBytes := []byte{
			byte(wantRegs[i*2] >> 8), byte(wantRegs[i*2]),
			byte(wantRegs[i*2+1] >> 8), byte(wantRegs[i*2+1]),
		}
		if string(got.Bytes) != string(wantGroupBytes) {
			t.Fatalf("results[%d].bytes = % X, want % X", i, got.Bytes, wantGroupBytes)
		}
	}
}

// 设备抓包风格的全链路闭环：设备读回的原始字节 → 寄存器 → 解码值 →
// 再编码 → 寄存器 → 原始字节，两端字节必须逐字节相同。
// 与只从编码出发的往返测试互补，这里假设的是"字节已经存在"。
func TestConvert32DeviceFrameRoundTrip(t *testing.T) {
	// 三个 float32 按 CDAB 字序存放在设备上：
	// 220.5 → 0x435C8000，0.12 → 0x3DF5C28F，-12.75 → 0xC14C0000。
	// 每寄存器线上固定高字节先发，所以整帧字节就是寄存器的大端展开。
	frame := []byte{
		0x80, 0x00, 0x43, 0x5C,
		0xC2, 0x8F, 0x3D, 0xF5,
		0x00, 0x00, 0xC1, 0x4C,
	}

	// 1) 字节 → 寄存器：每 2 字节一个大端寄存器，即设备原始地址顺序。
	regs := make([]uint32, 0, len(frame)/2)
	for i := 0; i < len(frame); i += 2 {
		regs = append(regs, uint32(frame[i])<<8|uint32(frame[i+1]))
	}

	// 2) 解码：值由 word_order 解释，字节/十六进制必须原样回显整帧。
	dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: regs,
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB,
	})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decRes.Results) != 3 {
		t.Fatalf("decode results = %d, want 3", len(decRes.Results))
	}
	wantValues := []float64{220.5, float64(float32(0.12)), -12.75}
	wantBytes := [][]byte{
		{0x80, 0x00, 0x43, 0x5C},
		{0xC2, 0x8F, 0x3D, 0xF5},
		{0x00, 0x00, 0xC1, 0x4C},
	}
	for i, want := range wantValues {
		got := decRes.Results[i]
		if got.Value != want {
			t.Fatalf("results[%d].value = %v, want %v", i, got.Value, want)
		}
		if string(got.Bytes) != string(wantBytes[i]) {
			t.Fatalf("results[%d].bytes = % X, want % X（应原样回显设备字节）", i, got.Bytes, wantBytes[i])
		}
	}

	// 3) 再编码：解码出的值按同字序写回，应还原出同一帧字节。
	enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
	encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
		Values:    wantValues,
		DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB,
	})
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	var back []byte
	for _, r := range encRes.Results {
		back = append(back, r.Bytes...)
	}
	if string(back) != string(frame) {
		t.Fatalf("重新编码的字节 = % X, want % X（闭环应逐字节还原）", back, frame)
	}
}

// 同一组寄存器只换 word_order 时，bytes/hex 必须逐位不变，只有 value 变。
// 这条用例锁定"bytes/hex 是设备报文的影子、不参与字序重排"的语义。
func TestConvert32WordOrderOnlyAffectsValue(t *testing.T) {
	regs := []uint32{0x7856, 0x3412}
	wantBytes := []byte{0x78, 0x56, 0x34, 0x12}
	const wantHex = "0x7856 0x3412"

	cases := []struct {
		order     bridgemodbus.WordOrder
		wantValue uint32
	}{
		{bridgemodbus.WordOrder_WORD_ORDER_ABCD, 0x78563412},
		{bridgemodbus.WordOrder_WORD_ORDER_CDAB, 0x34127856},
		{bridgemodbus.WordOrder_WORD_ORDER_BADC, 0x56781234},
		{bridgemodbus.WordOrder_WORD_ORDER_DCBA, 0x12345678},
	}
	l := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	for _, c := range cases {
		t.Run(c.order.String(), func(t *testing.T) {
			res, err := l.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
				Registers: regs,
				DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
				WordOrder: c.order,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := res.Results[0]
			if got.Value != float64(c.wantValue) || got.Uint32Value != c.wantValue {
				t.Fatalf("value = %v / uint32Value = %#x, want %#x",
					got.Value, got.Uint32Value, c.wantValue)
			}
			if got.Hex != wantHex {
				t.Fatalf("hex = %q, want %q（不应随字序变化）", got.Hex, wantHex)
			}
			if string(got.Bytes) != string(wantBytes) {
				t.Fatalf("bytes = % X, want % X（不应随字序变化）", got.Bytes, wantBytes)
			}
		})
	}
}

// 整数类型编码后同样应按字序解码还原，避免符号位或字序处理回归。
func TestConvert32IntRoundTrip(t *testing.T) {
	values := []float64{-12, 2147483647, -2147483648}
	for _, order := range []bridgemodbus.WordOrder{
		bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		bridgemodbus.WordOrder_WORD_ORDER_CDAB,
		bridgemodbus.WordOrder_WORD_ORDER_BADC,
		bridgemodbus.WordOrder_WORD_ORDER_DCBA,
	} {
		enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
		encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    values,
			DataType:  bridgemodbus.Value32Type_VALUE32_INT32,
			WordOrder: order,
		})
		if err != nil {
			t.Fatalf("wordOrder %v encode error: %v", order, err)
		}
		dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
		decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
			Registers: encRes.Registers,
			DataType:  bridgemodbus.Value32Type_VALUE32_INT32,
			WordOrder: order,
		})
		if err != nil {
			t.Fatalf("wordOrder %v decode error: %v", order, err)
		}
		for i, want := range values {
			if decRes.Results[i].Value != want || decRes.Results[i].Int32Value != int32(want) {
				t.Fatalf("wordOrder %v results[%d] = value %v / int32 %d, want %v",
					order, i, decRes.Results[i].Value, decRes.Results[i].Int32Value, want)
			}
		}
	}
}

// 单次条目数不设上限：超过历史上 1000 的内部上限也应正常编码、解码。
func TestConvert32NoItemLimit(t *testing.T) {
	const n = 1500
	values := make([]float64, n)
	for i := range values {
		values[i] = float64(i)
	}

	enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
	encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
		Values:    values,
		DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if len(encRes.Results) != n || len(encRes.Registers) != n*2 {
		t.Fatalf("encode = %d results / %d registers, want %d / %d",
			len(encRes.Results), len(encRes.Registers), n, n*2)
	}

	dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
	decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
		Registers: encRes.Registers,
		DataType:  bridgemodbus.Value32Type_VALUE32_UINT32,
		WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
	})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decRes.Results) != n {
		t.Fatalf("decode results = %d, want %d", len(decRes.Results), n)
	}
	if got := decRes.Results[n-1].Value; got != float64(n-1) {
		t.Fatalf("last value = %v, want %v", got, float64(n-1))
	}
}

// wordOrderExpect 是"某个字序应编出的两个寄存器"的期望值。
type wordOrderExpect struct {
	order bridgemodbus.WordOrder
	regs  [2]uint32
}

// 接口层按 4 字序 × 3 类型逐字序钉死寄存器、字节、十六进制与解码还原。
// bytex 单测已覆盖底层映射，这里保证 proto 字序转换与报文组装的接线在每种组合下都对。
func TestConvert32AllWordOrders(t *testing.T) {
	cases := []struct {
		name     string
		dataType bridgemodbus.Value32Type
		value    float64
		want     []wordOrderExpect
	}{
		{
			name:     "uint32 0x12345678",
			dataType: bridgemodbus.Value32Type_VALUE32_UINT32,
			value:    0x12345678,
			want: []wordOrderExpect{
				{bridgemodbus.WordOrder_WORD_ORDER_ABCD, [2]uint32{0x1234, 0x5678}},
				{bridgemodbus.WordOrder_WORD_ORDER_CDAB, [2]uint32{0x5678, 0x1234}},
				{bridgemodbus.WordOrder_WORD_ORDER_BADC, [2]uint32{0x3412, 0x7856}},
				{bridgemodbus.WordOrder_WORD_ORDER_DCBA, [2]uint32{0x7856, 0x3412}},
			},
		},
		{
			name:     "int32 -12",
			dataType: bridgemodbus.Value32Type_VALUE32_INT32,
			value:    -12,
			want: []wordOrderExpect{
				{bridgemodbus.WordOrder_WORD_ORDER_ABCD, [2]uint32{0xFFFF, 0xFFF4}},
				{bridgemodbus.WordOrder_WORD_ORDER_CDAB, [2]uint32{0xFFF4, 0xFFFF}},
				{bridgemodbus.WordOrder_WORD_ORDER_BADC, [2]uint32{0xFFFF, 0xF4FF}},
				{bridgemodbus.WordOrder_WORD_ORDER_DCBA, [2]uint32{0xF4FF, 0xFFFF}},
			},
		},
		{
			name:     "float32 220.5",
			dataType: bridgemodbus.Value32Type_VALUE32_FLOAT32,
			value:    220.5,
			want: []wordOrderExpect{
				{bridgemodbus.WordOrder_WORD_ORDER_ABCD, [2]uint32{0x435C, 0x8000}},
				{bridgemodbus.WordOrder_WORD_ORDER_CDAB, [2]uint32{0x8000, 0x435C}},
				{bridgemodbus.WordOrder_WORD_ORDER_BADC, [2]uint32{0x5C43, 0x0080}},
				{bridgemodbus.WordOrder_WORD_ORDER_DCBA, [2]uint32{0x0080, 0x5C43}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
			dec := NewBatchConvertRegistersToValue32Logic(context.Background(), nil)
			for _, w := range c.want {
				t.Run(w.order.String(), func(t *testing.T) {
					encRes, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
						Values:    []float64{c.value},
						DataType:  c.dataType,
						WordOrder: w.order,
					})
					if err != nil {
						t.Fatalf("encode error: %v", err)
					}
					got := encRes.Results[0]
					if len(got.Registers) != 2 || got.Registers[0] != w.regs[0] || got.Registers[1] != w.regs[1] {
						t.Fatalf("registers = %#v, want %#v", got.Registers, w.regs)
					}
					if encRes.Registers[0] != w.regs[0] || encRes.Registers[1] != w.regs[1] {
						t.Fatalf("flatten registers = %#v, want %#v", encRes.Registers, w.regs)
					}
					wantHex := fmt.Sprintf("0x%04X 0x%04X", w.regs[0], w.regs[1])
					if got.Hex != wantHex {
						t.Fatalf("hex = %q, want %q", got.Hex, wantHex)
					}
					wantBytes := []byte{byte(w.regs[0] >> 8), byte(w.regs[0]), byte(w.regs[1] >> 8), byte(w.regs[1])}
					if string(got.Bytes) != string(wantBytes) {
						t.Fatalf("bytes = % X, want % X", got.Bytes, wantBytes)
					}

					decRes, err := dec.BatchConvertRegistersToValue32(&bridgemodbus.BatchConvertRegistersToValue32Req{
						Registers: encRes.Registers,
						DataType:  c.dataType,
						WordOrder: w.order,
					})
					if err != nil {
						t.Fatalf("decode error: %v", err)
					}
					wantValue := c.value
					if c.dataType == bridgemodbus.Value32Type_VALUE32_FLOAT32 {
						wantValue = float64(float32(c.value))
					}
					if decRes.Results[0].Value != wantValue {
						t.Fatalf("decoded value = %v, want %v", decRes.Results[0].Value, wantValue)
					}
					if decRes.Results[0].Hex != wantHex {
						t.Fatalf("decoded hex = %q, want %q（应与编码侧一致）", decRes.Results[0].Hex, wantHex)
					}
					if string(decRes.Results[0].Bytes) != string(wantBytes) {
						t.Fatalf("decoded bytes = % X, want % X（应原样回显）", decRes.Results[0].Bytes, wantBytes)
					}
				})
			}
		})
	}
}

// float32 上界以转换结果为准：文档常标称的 3.4028235e38 转成 float32 仍是有限的
// MaxFloat32，必须接受；真正溢出为 ±Inf 或 NaN 的输入才拒绝。
func TestConvertValue32Float32RangeBoundary(t *testing.T) {
	enc := NewBatchConvertValue32ToRegistersLogic(context.Background(), nil)
	accepted := []float64{math.MaxFloat32, 3.4028235e38, -3.4028235e38}
	for _, v := range accepted {
		res, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{v},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		})
		if err != nil {
			t.Fatalf("%v 应被接受: %v", v, err)
		}
		if got, want := res.Results[0].Value, float64(float32(v)); got != want {
			t.Fatalf("回显 value = %v, want %v", got, want)
		}
	}

	rejected := []float64{math.Inf(1), math.Inf(-1), math.NaN(), 1e39, -1e39}
	for _, v := range rejected {
		if _, err := enc.BatchConvertValue32ToRegisters(&bridgemodbus.BatchConvertValue32ToRegistersReq{
			Values:    []float64{v},
			DataType:  bridgemodbus.Value32Type_VALUE32_FLOAT32,
			WordOrder: bridgemodbus.WordOrder_WORD_ORDER_ABCD,
		}); err == nil {
			t.Fatalf("%v 应被拒绝", v)
		}
	}
}
