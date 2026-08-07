package djisdk

import (
	"encoding/json"
	"testing"
)

func TestParseDeviceTypeAndRegistry(t *testing.T) {
	tests := []struct {
		raw      string
		wantType DeviceType
		wantName string
	}{
		{raw: "0-67-0", wantType: DeviceType{Domain: DeviceDomainAircraft, Type: 67, SubType: 0}, wantName: "Matrice 30"},
		{raw: "0-103-0", wantType: DeviceType{Domain: DeviceDomainAircraft, Type: 103, SubType: 0}, wantName: "Matrice 400"},
		{raw: "1-83-0", wantType: DeviceType{Domain: DeviceDomainPayload, Type: 83, SubType: 0}, wantName: "禅思 H30T"},
		{raw: "2-174-0", wantType: DeviceType{Domain: DeviceDomainRemoteController, Type: 174, SubType: 0}, wantName: "DJI RC Plus 2"},
		{raw: "3-3-0", wantType: DeviceType{Domain: DeviceDomainDock, Type: 3, SubType: 0}, wantName: "大疆机场 3"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseDeviceType(tt.raw)
			if err != nil || got != tt.wantType || got.String() != tt.raw {
				t.Fatalf("ParseDeviceType(%q) = %+v, %v", tt.raw, got, err)
			}
			name, ok := LookupDeviceTypeName(tt.raw)
			if !ok || name != tt.wantName {
				t.Fatalf("LookupDeviceTypeName(%q) = %q, %v, want %q", tt.raw, name, ok, tt.wantName)
			}
		})
	}
}

func TestParseDeviceTypeRejectsNonOfficialFormats(t *testing.T) {
	for _, raw := range []string{"", "dock", "remote", "0-67", "0-x-1", "0-67-0-1", "-1-67-0", "4-1-0", " 0-67-0"} {
		if _, err := ParseDeviceType(raw); err == nil {
			t.Errorf("ParseDeviceType(%q) succeeded, want error", raw)
		}
	}
	if name, ok := LookupDeviceTypeName("0-999-0"); ok || name != "" {
		t.Fatalf("LookupDeviceTypeName(unknown) = %q, %v", name, ok)
	}
}

func TestDeviceTypeRegistryMatchesOfficialProductList(t *testing.T) {
	wants := map[DeviceType]string{
		{Domain: DeviceDomainAircraft, Type: 103, SubType: 0}:         "Matrice 400",
		{Domain: DeviceDomainAircraft, Type: 89, SubType: 0}:          "Matrice 350 RTK",
		{Domain: DeviceDomainAircraft, Type: 60, SubType: 0}:          "Matrice 300 RTK",
		{Domain: DeviceDomainAircraft, Type: 67, SubType: 0}:          "Matrice 30",
		{Domain: DeviceDomainAircraft, Type: 67, SubType: 1}:          "Matrice 30T",
		{Domain: DeviceDomainAircraft, Type: 77, SubType: 0}:          "Mavic 3 行业系列（M3E 相机）",
		{Domain: DeviceDomainAircraft, Type: 77, SubType: 1}:          "Mavic 3 行业系列（M3T 相机）",
		{Domain: DeviceDomainAircraft, Type: 77, SubType: 3}:          "Mavic 3 行业系列（M3TA 相机）",
		{Domain: DeviceDomainAircraft, Type: 91, SubType: 0}:          "Matrice 3D",
		{Domain: DeviceDomainAircraft, Type: 91, SubType: 1}:          "Matrice 3TD",
		{Domain: DeviceDomainAircraft, Type: 100, SubType: 0}:         "Matrice 4D",
		{Domain: DeviceDomainAircraft, Type: 100, SubType: 1}:         "Matrice 4TD",
		{Domain: DeviceDomainAircraft, Type: 99, SubType: 0}:          "DJI Matrice 4 系列（M4E 相机）",
		{Domain: DeviceDomainAircraft, Type: 99, SubType: 1}:          "DJI Matrice 4 系列（M4T 相机）",
		{Domain: DeviceDomainPayload, Type: 39, SubType: 0}:           "飞行器 FPV 相机",
		{Domain: DeviceDomainPayload, Type: 176, SubType: 0}:          "飞行器辅助影像",
		{Domain: DeviceDomainPayload, Type: 20, SubType: 0}:           "禅思 Z30",
		{Domain: DeviceDomainPayload, Type: 26, SubType: 0}:           "禅思 XT2",
		{Domain: DeviceDomainPayload, Type: 41, SubType: 0}:           "禅思 XTS",
		{Domain: DeviceDomainPayload, Type: 42, SubType: 0}:           "禅思 H20",
		{Domain: DeviceDomainPayload, Type: 43, SubType: 0}:           "禅思 H20T",
		{Domain: DeviceDomainPayload, Type: 61, SubType: 0}:           "禅思 H20N",
		{Domain: DeviceDomainPayload, Type: 82, SubType: 0}:           "禅思 H30",
		{Domain: DeviceDomainPayload, Type: 83, SubType: 0}:           "禅思 H30T",
		{Domain: DeviceDomainPayload, Type: 52, SubType: 0}:           "Matrice 30 相机",
		{Domain: DeviceDomainPayload, Type: 53, SubType: 0}:           "Matrice 30T 相机",
		{Domain: DeviceDomainPayload, Type: 88, SubType: 0}:           "DJI Matrice 4E 相机",
		{Domain: DeviceDomainPayload, Type: 89, SubType: 0}:           "DJI Matrice 4T 相机",
		{Domain: DeviceDomainPayload, Type: 66, SubType: 0}:           "Mavic 3E 相机",
		{Domain: DeviceDomainPayload, Type: 67, SubType: 0}:           "Mavic 3T 相机",
		{Domain: DeviceDomainPayload, Type: 129, SubType: 0}:          "Mavic 3TA 相机",
		{Domain: DeviceDomainPayload, Type: 80, SubType: 0}:           "Matrice 3D 相机",
		{Domain: DeviceDomainPayload, Type: 81, SubType: 0}:           "Matrice 3TD 相机",
		{Domain: DeviceDomainPayload, Type: 98, SubType: 0}:           "Matrice 4D 相机",
		{Domain: DeviceDomainPayload, Type: 99, SubType: 0}:           "Matrice 4TD 相机",
		{Domain: DeviceDomainPayload, Type: 165, SubType: 0}:          "大疆机场相机",
		{Domain: DeviceDomainRemoteController, Type: 56, SubType: 0}:  "DJI 带屏遥控器行业版",
		{Domain: DeviceDomainRemoteController, Type: 119, SubType: 0}: "DJI RC Plus",
		{Domain: DeviceDomainRemoteController, Type: 174, SubType: 0}: "DJI RC Plus 2",
		{Domain: DeviceDomainRemoteController, Type: 144, SubType: 0}: "DJI RC Pro 行业版",
		{Domain: DeviceDomainDock, Type: 1, SubType: 0}:               "大疆机场",
		{Domain: DeviceDomainDock, Type: 2, SubType: 0}:               "大疆机场 2",
		{Domain: DeviceDomainDock, Type: 3, SubType: 0}:               "大疆机场 3",
	}

	if len(deviceTypeRegistry) != len(wants) {
		t.Fatalf("deviceTypeRegistry has %d entries, want %d", len(deviceTypeRegistry), len(wants))
	}
	for deviceType, wantName := range wants {
		definition, ok := LookupDeviceType(deviceType)
		if !ok || definition.DeviceType != deviceType || definition.Name != wantName {
			t.Errorf("LookupDeviceType(%s) = %+v, %v, want name %q", deviceType, definition, ok, wantName)
		}
	}
}

func TestHmsArgStringPreservesOfficialAlarmID(t *testing.T) {
	args := HmsArgs{"alarmid": "0x00AbCdEf", "numeric": json.Number("42")}
	if got, ok := args.String("alarmid"); !ok || got != "0x00AbCdEf" {
		t.Fatalf("args.String(alarmid) = %q, %v", got, ok)
	}
	if got, ok := args.String("numeric"); !ok || got != "42" {
		t.Fatalf("args.String(numeric) = %q, %v", got, ok)
	}
	if _, ok := args.String("missing"); ok {
		t.Fatal("args.String(missing) succeeded")
	}
}

func TestHmsArgsIntStrict(t *testing.T) {
	args := HmsArgs{
		"integer":         float64(7),
		"integer_decimal": float64(7.0),
		"fraction":        1.5,
		"overflow":        uint64(^uint(0)),
		"number_decimal":  json.Number("2.5"),
		"nil":             nil,
		"unsupported":     true,
	}
	if got, ok := args.Int("integer"); !ok || got != 7 {
		t.Fatalf("args.Int(integer) = %d, %v", got, ok)
	}
	if got, ok := args.Int("integer_decimal"); !ok || got != 7 {
		t.Fatalf("args.Int(integer_decimal) = %d, %v", got, ok)
	}
	for _, name := range []string{"fraction", "overflow", "number_decimal", "nil", "unsupported", "missing"} {
		if got, ok := args.Int(name); ok {
			t.Errorf("args.Int(%s) = %d, true; want rejected", name, got)
		}
	}
}

func TestPayloadPlacementIsSeparateFromDeviceType(t *testing.T) {
	sharedFPV, ok := LookupDeviceType(DeviceType{Domain: DeviceDomainPayload, Type: 39, SubType: 0})
	if !ok || sharedFPV.Name != "飞行器 FPV 相机" {
		t.Fatalf("shared FPV definition = %+v, %v", sharedFPV, ok)
	}

	wants := map[PayloadGimbalIndex]PayloadGimbalPosition{
		PayloadGimbalMainOrLowerLeft: "主云台（M300 RTK 为机身下方左云台）",
		PayloadGimbalLowerRight:      "机身下方右云台（M300 RTK）",
		PayloadGimbalUpper:           "机身上方云台（M300 RTK）",
		PayloadGimbalFPV:             "FPV 相机",
	}
	for index, want := range wants {
		definition, position, ok := DescribePayload(PayloadPlacement{DeviceType: sharedFPV.DeviceType, GimbalIndex: index})
		if !ok || definition != sharedFPV || position != want {
			t.Errorf("DescribePayload(index=%d) = %+v, %q, %v", index, definition, position, ok)
		}
	}
}
