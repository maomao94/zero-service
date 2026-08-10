package logic

import (
	"database/sql"
	"testing"

	"zero-service/app/djicloud/model/gormmodel"
)

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
