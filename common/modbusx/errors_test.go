package modbusx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grid-x/modbus"
)

func TestException(t *testing.T) {
	err := fmt.Errorf("read failed: %w", &modbus.Error{
		FunctionCode:  0x84,
		ExceptionCode: modbus.ExceptionCodeIllegalDataAddress,
	})

	functionCode, exceptionCode, ok := Exception(err)
	if !ok {
		t.Fatalf("Exception() ok = false, want true")
	}
	if functionCode != 0x84 {
		t.Fatalf("functionCode = %#x, want 0x84", functionCode)
	}
	if exceptionCode != modbus.ExceptionCodeIllegalDataAddress {
		t.Fatalf("exceptionCode = %#x, want %#x", exceptionCode, modbus.ExceptionCodeIllegalDataAddress)
	}
}

func TestExceptionNotModbusError(t *testing.T) {
	if _, _, ok := Exception(errors.New("boom")); ok {
		t.Fatalf("Exception() ok = true, want false")
	}
}

func TestExceptionMessage(t *testing.T) {
	if got := ExceptionMessage(modbus.ExceptionCodeIllegalDataAddress); got != "非法数据地址" {
		t.Fatalf("ExceptionMessage() = %q", got)
	}
	if got := ExceptionMessage(0x7F); got != "未知异常" {
		t.Fatalf("ExceptionMessage(unknown) = %q", got)
	}
}

func TestIsTimeout(t *testing.T) {
	if !IsTimeout(context.DeadlineExceeded) {
		t.Fatalf("IsTimeout(DeadlineExceeded) = false, want true")
	}
	if IsTimeout(errors.New("boom")) {
		t.Fatalf("IsTimeout(internal error) = true, want false")
	}
}

func TestIsStreamDesynced(t *testing.T) {
	err := fmt.Errorf("read response: %w", modbus.ErrStreamDesynced)
	if !IsStreamDesynced(err) {
		t.Fatalf("IsStreamDesynced(wrapped) = false, want true")
	}
	if IsStreamDesynced(errors.New("boom")) {
		t.Fatalf("IsStreamDesynced(internal error) = true, want false")
	}
}
