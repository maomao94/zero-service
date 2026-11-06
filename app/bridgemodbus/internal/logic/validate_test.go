package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/errx"
	"zero-service/common/modbusx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/grid-x/modbus"
)

func TestWrapModbusWriteErrorUnknownOutcome(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code extproto.Code
	}{
		{"desynced", modbus.ErrStreamDesynced, extproto.Code__1_00_THIRD_PARTY},
		{"timeout", context.DeadlineExceeded, extproto.Code__1_00_TIMEOUT},
		{"disconnected", io.EOF, extproto.Code__1_00_THIRD_PARTY},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := errx.CodeFromStatus(wrapModbusWriteError("写寄存器", tt.err))
			if code != int(tt.code) || !strings.Contains(msg, "写入执行结果未知，请核实设备状态") {
				t.Fatalf("code=%d msg=%q", code, msg)
			}
		})
	}
	if got := wrapModbusWriteError("写寄存器", context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatalf("canceled = %v", got)
	}
	if got := wrapModbusWriteError("写寄存器", nil); got != nil {
		t.Fatalf("nil error = %v", got)
	}
	exception := &modbus.Error{FunctionCode: 0x86, ExceptionCode: modbus.ExceptionCodeIllegalDataAddress}
	code, msg := errx.CodeFromStatus(wrapModbusWriteError("写寄存器", exception))
	if code != int(extproto.Code__1_00_THIRD_PARTY_API) || strings.Contains(msg, "执行结果未知") {
		t.Fatalf("exception code=%d msg=%q", code, msg)
	}
}

// 客户端失效发生在进入 SDK 之前，写请求必须提示"未执行"而不是"结果未知"。
func TestWrapModbusErrorClientInvalidated(t *testing.T) {
	for _, write := range []bool{true, false} {
		wrap := wrapModbusError
		if write {
			wrap = wrapModbusWriteError
		}
		wrapped := wrap("写单个寄存器", modbusx.ErrClientInvalidated)
		code, msg := errx.CodeFromStatus(wrapped)
		if code != int(extproto.Code__1_00_BIZ) {
			t.Fatalf("write=%v code = %d, want %d", write, code, extproto.Code__1_00_BIZ)
		}
		if !strings.Contains(msg, "未执行") || strings.Contains(msg, "执行结果未知") {
			t.Fatalf("write=%v msg = %q", write, msg)
		}
		if !errors.Is(wrapped, modbusx.ErrClientInvalidated) {
			t.Fatalf("write=%v cause lost: %v", write, wrapped)
		}
	}
}

func TestValidateAddress(t *testing.T) {
	address, err := validateAddress("address", 0)
	if err != nil || address != 0 {
		t.Fatalf("validateAddress(0) = %d, %v", address, err)
	}
	address, err = validateAddress("address", 0xFFFF)
	if err != nil || address != 0xFFFF {
		t.Fatalf("validateAddress(65535) = %d, %v", address, err)
	}

	if _, err := validateAddress("address", 0x10000); err == nil {
		t.Fatalf("validateAddress(65536) error = nil, want error")
	} else if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
		t.Fatalf("validateAddress(65536) code = %d, want %d", code, extproto.Code__1_00_PARAM_INVALID)
	}
}

func TestValidateQuantity(t *testing.T) {
	if _, err := validateQuantity("quantity", 1, maxReadRegisters); err != nil {
		t.Fatalf("validateQuantity(1) error = %v", err)
	}
	if _, err := validateQuantity("quantity", maxReadRegisters, maxReadRegisters); err != nil {
		t.Fatalf("validateQuantity(max) error = %v", err)
	}
	if _, err := validateQuantity("quantity", 0, maxReadRegisters); err == nil {
		t.Fatalf("validateQuantity(0) error = nil, want error")
	}
	if _, err := validateQuantity("quantity", maxReadRegisters+1, maxReadRegisters); err == nil {
		t.Fatalf("validateQuantity(max+1) error = nil, want error")
	}
}

func TestValidateReadDeviceIDCode(t *testing.T) {
	for _, code := range []uint32{1, 2, 3} {
		got, err := validateReadDeviceIDCode(code)
		if err != nil {
			t.Fatalf("validateReadDeviceIDCode(%d) error = %v", code, err)
		}
		if uint32(got) != code {
			t.Fatalf("validateReadDeviceIDCode(%d) = %d", code, got)
		}
	}

	// 0x04（指定对象）必须走 ReadDeviceIdentificationSpecificObject，
	// grid-x v1.5.1 的 ReadDeviceIdentification 不支持该值。
	for _, code := range []uint32{0, 4, 5, 255} {
		_, err := validateReadDeviceIDCode(code)
		if err == nil {
			t.Fatalf("validateReadDeviceIDCode(%d) error = nil, want error", code)
		}
		if got, _ := errx.CodeFromStatus(err); got != int(extproto.Code__1_00_PARAM_INVALID) {
			t.Fatalf("validateReadDeviceIDCode(%d) code = %d, want %d",
				code, got, extproto.Code__1_00_PARAM_INVALID)
		}
	}
}

func TestValidate16BitValue(t *testing.T) {
	if _, err := validate16BitValue("value", 0xFFFF); err != nil {
		t.Fatalf("validate16BitValue(65535) error = %v", err)
	}
	if _, err := validate16BitValue("value", 0x10000); err == nil {
		t.Fatalf("validate16BitValue(65536) error = nil, want error")
	}
}

func TestValidateValuesCount(t *testing.T) {
	if err := validateValuesCount("quantity", 2, 2); err != nil {
		t.Fatalf("validateValuesCount(2,2) error = %v", err)
	}
	if err := validateValuesCount("quantity", 2, 1); err == nil {
		t.Fatalf("validateValuesCount(2,1) error = nil, want error")
	}
}

func TestWrapModbusErrorException(t *testing.T) {
	err := &modbus.Error{
		FunctionCode:  0x84,
		ExceptionCode: modbus.ExceptionCodeIllegalDataAddress,
	}

	wrapped := wrapModbusError("读取输入寄存器", err)
	code, msg := errx.CodeFromStatus(wrapped)
	if code != int(extproto.Code__1_00_THIRD_PARTY_API) {
		t.Fatalf("code = %d, want %d", code, extproto.Code__1_00_THIRD_PARTY_API)
	}

	vendor, vendorCode, message, ok := tool.ParseThirdPartyAPIError(msg)
	if !ok {
		t.Fatalf("ParseThirdPartyAPIError(%q) ok = false", msg)
	}
	if vendor != "modbus" {
		t.Fatalf("vendor = %q, want modbus", vendor)
	}
	if vendorCode != "0x02" {
		t.Fatalf("vendor code = %q, want 0x02", vendorCode)
	}
	if !strings.Contains(message, "非法数据地址") {
		t.Fatalf("message = %q, want 非法数据地址", message)
	}
}

func TestWrapModbusErrorTimeout(t *testing.T) {
	wrapped := wrapModbusError("读取保持寄存器", context.DeadlineExceeded)
	code, _ := errx.CodeFromStatus(wrapped)
	if code != int(extproto.Code__1_00_TIMEOUT) {
		t.Fatalf("code = %d, want %d", code, extproto.Code__1_00_TIMEOUT)
	}
}

func TestWrapModbusErrorStreamDesynced(t *testing.T) {
	wrapped := wrapModbusError("写入寄存器", fmt.Errorf("read: %w", modbus.ErrStreamDesynced))
	code, msg := errx.CodeFromStatus(wrapped)
	if code != int(extproto.Code__1_00_THIRD_PARTY) {
		t.Fatalf("code = %d, want %d", code, extproto.Code__1_00_THIRD_PARTY)
	}
	if !strings.Contains(msg, "请重新发起读取") {
		t.Fatalf("msg = %q, want retry hint", msg)
	}
}

func TestWrapModbusErrorGeneric(t *testing.T) {
	wrapped := wrapModbusError("写入线圈", errors.New("connection refused"))
	code, _ := errx.CodeFromStatus(wrapped)
	if code != int(extproto.Code__1_00_THIRD_PARTY) {
		t.Fatalf("code = %d, want %d", code, extproto.Code__1_00_THIRD_PARTY)
	}
}

// 十进制写入先在转换阶段校验范围，错误码必须与写入校验一致（100103）。
func TestWriteWithDecimalOutOfRangeUsesParamInvalidCode(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func() error
	}{
		{"single unsigned", func() error {
			_, err := NewWriteSingleRegisterWithDecimalLogic(context.Background(), nil).
				WriteSingleRegisterWithDecimal(&bridgemodbus.WriteSingleRegisterWithDecimalReq{
					Value: 70000, Unsigned: true,
				})
			return err
		}},
		{"multiple signed", func() error {
			_, err := NewWriteMultipleRegistersWithDecimalLogic(context.Background(), nil).
				WriteMultipleRegistersWithDecimal(&bridgemodbus.WriteMultipleRegistersWithDecimalReq{
					Quantity: 1, Values: []int32{70000},
				})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
				t.Fatalf("code = %d, want %d: %v", code, extproto.Code__1_00_PARAM_INVALID, err)
			}
		})
	}
}
