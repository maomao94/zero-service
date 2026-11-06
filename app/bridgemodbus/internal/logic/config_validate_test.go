package logic

import (
	"strings"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

func TestValidateSaveConfig(t *testing.T) {
	for _, tt := range []struct {
		name    string
		code    string
		address string
		slave   uint32
		invalid bool
	}{
		{"valid", "pump", "192.0.2.1:502", 1, false},
		{"hostname", "pump", "modbus.local:502", 247, false},
		{"ipv6", "pump", "[2001:db8::1]:65535", 1, false},
		{"max code", strings.Repeat("中", 128), "localhost:1", 1, false},
		{"empty code", "", "localhost:502", 1, true},
		{"blank code", " \t\n", "localhost:502", 1, true},
		{"long code", strings.Repeat("中", 129), "localhost:502", 1, true},
		{"empty address", "pump", "", 1, true},
		{"blank address", "pump", " \t", 1, true},
		{"long address", "pump", strings.Repeat("a", 61) + ":502", 1, true},
		{"missing port", "pump", "localhost", 1, true},
		{"missing host", "pump", ":502", 1, true},
		{"zero port", "pump", "localhost:0", 1, true},
		{"large port", "pump", "localhost:65536", 1, true},
		{"named port", "pump", "localhost:modbus", 1, true},
		{"signed port", "pump", "localhost:+502", 1, true},
		{"url", "pump", "tcp://localhost:502", 1, true},
		{"space in host", "pump", "local host:502", 1, true},
		{"zero slave", "pump", "localhost:502", 0, true},
		{"reserved slave", "pump", "localhost:502", 248, true},
		{"overflow slave", "pump", "localhost:502", ^uint32(0), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSaveConfig(&bridgemodbus.SaveConfigReq{ModbusCode: tt.code, SlaveAddress: tt.address, Slave: tt.slave})
			if tt.invalid {
				assertConfigParamError(t, err)
			} else if err != nil {
				t.Fatalf("validateSaveConfig() = %v", err)
			}
		})
	}
}

func TestValidateConfigBatch(t *testing.T) {
	for _, tt := range []struct {
		name    string
		values  []string
		invalid bool
	}{
		{"empty", nil, true},
		{"empty item", []string{"pump", ""}, true},
		{"blank item", []string{"pump", " \t"}, true},
		{"long item", []string{strings.Repeat("a", 129)}, true},
		{"invalid utf8", []string{"\xff"}, true},
		{"valid", []string{"pump"}, false},
		{"max count", repeatConfigCodes(maxConfigBatchSize), false},
		{"too many", repeatConfigCodes(maxConfigBatchSize + 1), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigBatch("modbusCode", tt.values, 128)
			if tt.invalid {
				assertConfigParamError(t, err)
			} else if err != nil {
				t.Fatalf("validateConfigBatch() = %v", err)
			}
		})
	}
}

// 使用没有数据库的 Logic，确认非法参数在访问数据库之前被拒绝。
func TestConfigValidationBeforeDB(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func() error
	}{
		{"save nil", func() error { _, err := (&SaveConfigLogic{}).SaveConfig(nil); return err }},
		{"save empty", func() error { _, err := (&SaveConfigLogic{}).SaveConfig(&bridgemodbus.SaveConfigReq{}); return err }},
		{"get nil", func() error { _, err := (&GetConfigByCodeLogic{}).GetConfigByCode(nil); return err }},
		{"get blank", func() error {
			_, err := (&GetConfigByCodeLogic{}).GetConfigByCode(&bridgemodbus.GetConfigByCodeReq{ModbusCode: " "})
			return err
		}},
		{"batch nil", func() error { _, err := (&BatchGetConfigByCodeLogic{}).BatchGetConfigByCode(nil); return err }},
		{"batch blank", func() error {
			_, err := (&BatchGetConfigByCodeLogic{}).BatchGetConfigByCode(&bridgemodbus.BatchGetConfigByCodeReq{ModbusCode: []string{""}})
			return err
		}},
		{"delete nil", func() error { _, err := (&DeleteConfigLogic{}).DeleteConfig(nil); return err }},
		{"delete blank", func() error {
			_, err := (&DeleteConfigLogic{}).DeleteConfig(&bridgemodbus.DeleteConfigReq{Ids: []string{" "}})
			return err
		}},
		{"delete long id", func() error {
			_, err := (&DeleteConfigLogic{}).DeleteConfig(&bridgemodbus.DeleteConfigReq{Ids: []string{strings.Repeat("a", 37)}})
			return err
		}},
		{"page nil", func() error { _, err := (&PageListConfigLogic{}).PageListConfig(nil); return err }},
		{"page status", func() error {
			_, err := (&PageListConfigLogic{}).PageListConfig(&bridgemodbus.PageListConfigReq{Status: 3})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) { assertConfigParamError(t, tt.call()) })
	}
}

func repeatConfigCodes(count int) []string {
	values := make([]string, count)
	for i := range values {
		values[i] = "pump"
	}
	return values
}

func assertConfigParamError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected parameter error")
	}
	if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
		t.Fatalf("error code = %d, want %d: %v", code, extproto.Code__1_00_PARAM_INVALID, err)
	}
}
