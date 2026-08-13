package logic

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
)

func TestCommandResWrapsOnlyDJIError(t *testing.T) {
	djiErr := djisdk.NewDJIError(123)
	res, err := commandRes("tid-1", fmt.Errorf("device rejected command: %w", djiErr))
	if err != nil {
		t.Fatalf("commandRes() error = %v, want nil", err)
	}
	if res == nil {
		t.Fatal("commandRes() response = nil")
	}
	if res.Code != -1 || res.Message != djiErr.Message || res.Tid != "tid-1" || res.ReasonCode != int32(djiErr.Code) {
		t.Fatalf("commandRes() response = %+v, want wrapped DJI error", res)
	}
}

func TestCommandResReturnsInfrastructureError(t *testing.T) {
	cause := errors.New("mqtt unavailable")
	res, err := commandRes("tid-2", cause)
	if res != nil {
		t.Fatalf("commandRes() response = %+v, want nil", res)
	}
	if err == nil {
		t.Fatal("commandRes() error = nil")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("commandRes() error = %v, want original cause", err)
	}
}

func TestToDeviceInfoReturnsPersistedDeviceTypeAndName(t *testing.T) {
	item := &gormmodel.DjiDevice{
		DeviceSn:     "drone-1",
		GatewaySn:    "dock-1",
		DeviceType:   "0-60-0",
		DeviceName:   "persisted product name",
		IsOnline:     true,
		LastOnlineAt: sql.NullTime{},
	}

	got := toDeviceInfo(item)
	if got.DeviceType != item.DeviceType || got.DeviceName != item.DeviceName {
		t.Fatalf("device type/name = %q/%q, want persisted %q/%q", got.DeviceType, got.DeviceName, item.DeviceType, item.DeviceName)
	}
}

func TestToTopoInfoListReturnsPersistedDeviceTypeAndName(t *testing.T) {
	items := []gormmodel.DjiDeviceTopo{{
		GatewaySn:        "dock-1",
		SubDeviceSn:      "drone-1",
		Domain:           "0",
		SubDeviceType:    60,
		SubDeviceSubType: 0,
		DeviceType:       "0-60-0",
		DeviceName:       "persisted product name",
	}}

	got := toTopoInfoList(items)
	if len(got) != 1 {
		t.Fatalf("topo info count = %d, want 1", len(got))
	}
	if got[0].DeviceType != items[0].DeviceType || got[0].DeviceName != items[0].DeviceName {
		t.Fatalf("topo device type/name = %q/%q, want persisted %q/%q", got[0].DeviceType, got[0].DeviceName, items[0].DeviceType, items[0].DeviceName)
	}
}

func TestDeviceRPCViewsReturnPersistedUnknownSentinels(t *testing.T) {
	device := &gormmodel.DjiDevice{
		DeviceSn:   "device-unknown",
		DeviceType: gormmodel.DjiDeviceUnknown,
		DeviceName: gormmodel.DjiDeviceUnknown,
	}
	deviceInfo := toDeviceInfo(device)
	if deviceInfo.DeviceType != gormmodel.DjiDeviceUnknown || deviceInfo.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("device RPC type/name = %q/%q, want persisted unknown sentinels", deviceInfo.DeviceType, deviceInfo.DeviceName)
	}

	topoInfo := toTopoInfoList([]gormmodel.DjiDeviceTopo{{
		GatewaySn:   "dock-unknown",
		SubDeviceSn: "device-unknown",
		DeviceType:  gormmodel.DjiDeviceUnknown,
		DeviceName:  gormmodel.DjiDeviceUnknown,
	}})
	if len(topoInfo) != 1 {
		t.Fatalf("topology RPC count = %d, want 1", len(topoInfo))
	}
	if topoInfo[0].DeviceType != gormmodel.DjiDeviceUnknown || topoInfo[0].DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("topology RPC type/name = %q/%q, want persisted unknown sentinels", topoInfo[0].DeviceType, topoInfo[0].DeviceName)
	}
}
