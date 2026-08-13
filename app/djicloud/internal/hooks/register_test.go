package hooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
	gooteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHookTestDB(t *testing.T) *gormx.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&loc=UTC"), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Unix(1710000000, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("open sqlite db error = %v", err)
	}
	if err := db.AutoMigrate(
		&gormmodel.DjiDevice{},
		&gormmodel.DjiDeviceTopo{},
		&gormmodel.DjiDeviceOsdSnapshot{},
		&gormmodel.DjiDeviceStateSnapshot{},
		&gormmodel.DjiDockFlightTask{},
		&gormmodel.DjiDockDeviceFlightTaskState{},
		&gormmodel.DjiFlightTaskReady{},
		&gormmodel.DjiRemoteLogEvent{},
		&gormmodel.DjiReturnHomeEvent{},
		&gormmodel.DjiDrcUpEvent{},
		&gormmodel.DjiHmsAlert{},
	); err != nil {
		t.Fatalf("auto migrate hook models error = %v", err)
	}

	return &gormx.DB{DB: db}
}

func TestRegisterDjiClientRegistersHandlersAndOnlineChecker(t *testing.T) {
	onlineCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache online error = %v", err)
	}
	db := newHookTestDB(t)
	handlerOpts := WithDjiClientOptions(RegisterDjiClientOptions{
		DB:          db,
		HmsResolver: djisdk.MustNewHmsResolver(djisdk.HmsConfig{}),
		OnlineCache: onlineCache,
	})
	allOpts := []djisdk.ClientOption{
		djisdk.WithPendingTTL(time.Second),
		djisdk.WithReplyConfig(djisdk.ReplyConfig{}),
	}
	allOpts = append(allOpts, handlerOpts...)
	client := djisdk.NewClient(nil, allOpts...)

	ctx := context.Background()
	statusPayload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"update_topo","data":{"sub_devices":[]}}`)
	if err := client.HandleStatus(ctx, statusPayload, djisdk.StatusTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleStatus() error = %v", err)
	}
	if IsOnline(onlineCache, "gateway-1") {
		t.Fatal("expected status handler not to refresh online cache")
	}
	osdPayload := []byte(`{"tid":"tid-osd","bid":"bid-osd","timestamp":1710000000000,"gateway":"gateway-1","data":{"mode_code":1}}`)
	if err := client.HandleOsd(ctx, osdPayload, djisdk.OsdTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleOsd() error = %v", err)
	}
	if !IsOnline(onlineCache, "gateway-1") {
		t.Fatal("expected osd handler to refresh online cache")
	}
	if _, err := client.SendCommand(ctx, "offline-gateway", djisdk.MethodReturnHome, nil); err == nil {
		t.Fatal("expected offline checker to reject unknown gateway")
	} else if err.Error() != "device offline: gateway_sn=offline-gateway command rejected" {
		t.Fatalf("SendCommand() error = %v, want offline checker rejection", err)
	}

	progressPayload := []byte(`{"tid":"tid-2","bid":"bid-2","gateway":"gateway-1","need_reply":0,"method":"flighttask_progress","data":{"result":0,"output":{"ext":{"flight_id":"flight-1","wayline_mission_state":1,"current_waypoint_index":2,"media_count":3,"track_id":"track-1"},"progress":{"current_step":2,"percent":50},"status":"ok"}}}`)
	if err := client.HandleEvents(ctx, progressPayload, djisdk.EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents() error = %v", err)
	}

	requestsPayload := []byte(`{"tid":"tid-3","bid":"bid-3","timestamp":1710000000000,"method":"airport_bind_status","data":{"status":1}}`)
	if err := client.HandleRequests(ctx, requestsPayload, "thing/product/gateway-1/requests", ""); err != nil {
		t.Fatalf("HandleRequests() error = %v", err)
	}

	drcPayload := []byte(`{"tid":"tid-drc","bid":"bid-drc","timestamp":1710000000000,"method":"stick_control","seq":1,"data":{"result":0,"output":{"seq":1}}}`)
	if err := client.HandleDrcUp(ctx, drcPayload, djisdk.DrcUpTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleDrcUp() error = %v", err)
	}
	var drcEvent struct {
		RawJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDrcUpEvent{}).Select("raw_json").Where("gateway_sn = ? AND method = ?", "gateway-1", djisdk.MethodStickControl).First(&drcEvent).Error; err != nil {
		t.Fatalf("find registered drc event error = %v", err)
	}
	if drcEvent.RawJSON != `{"result":0,"output":{"seq":1}}` {
		t.Fatalf("registered RawJSON = %q, want stick_control ack", drcEvent.RawJSON)
	}
}

func TestRegisterDjiClientWithoutOnlineCacheHandlesUpstreamWithoutOnlineChecker(t *testing.T) {
	handlerOpts := WithDjiClientOptions(RegisterDjiClientOptions{})
	allOpts := []djisdk.ClientOption{
		djisdk.WithPendingTTL(time.Second),
		djisdk.WithReplyConfig(djisdk.ReplyConfig{}),
	}
	allOpts = append(allOpts, handlerOpts...)
	client := djisdk.NewClient(nil, allOpts...)

	requestsPayload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"airport_bind_status","data":{"status":1}}`)
	if err := client.HandleRequests(context.Background(), requestsPayload, "thing/product/gateway-1/requests", ""); err != nil {
		t.Fatalf("HandleRequests() error = %v", err)
	}
}

func TestStateTelemetryUpdatesDeviceDataButNotOnline(t *testing.T) {
	db := newHookTestDB(t)
	onlineCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache online error = %v", err)
	}
	ctx := context.Background()

	NewStateTelemetryHandler(db, onlineCache, nil)(ctx, "drone-1", &djisdk.StateMessage{
		Gateway:   "dock-1",
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1, "firmware_version": "05.01.0214", "hardware_version": "M4D"},
	})

	if IsOnline(onlineCache, "drone-1") {
		t.Fatal("expected state telemetry not to refresh device online cache")
	}
	if IsOnline(onlineCache, "dock-1") {
		t.Fatal("expected state telemetry not to refresh gateway online cache")
	}
	var device struct {
		GatewaySn       string
		DeviceType      string
		DeviceName      string
		FirmwareVersion string
		HardwareVersion string
		IsOnline        bool
		LastOnlineAt    sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "drone-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-1" {
		t.Fatalf("GatewaySn = %s, want dock-1", device.GatewaySn)
	}
	if device.DeviceType != gormmodel.DjiDeviceUnknown || device.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("device type/name = %q/%q, want unknown sentinels", device.DeviceType, device.DeviceName)
	}
	if device.FirmwareVersion != "05.01.0214" || device.HardwareVersion != "M4D" {
		t.Fatalf("device versions = %s/%s, want 05.01.0214/M4D", device.FirmwareVersion, device.HardwareVersion)
	}
	if device.IsOnline {
		t.Fatal("expected state telemetry not to mark device online")
	}
	if device.LastOnlineAt.Valid {
		t.Fatal("expected state telemetry not to update last online time")
	}
	var snapshot struct {
		RawJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceStateSnapshot{}).Select("raw_json").Where("device_sn = ?", "drone-1").First(&snapshot).Error; err != nil {
		t.Fatalf("find state snapshot error = %v", err)
	}
	if !strings.Contains(snapshot.RawJSON, "firmware_version") {
		t.Fatalf("expected pushMode=1 property to be stored in state snapshot raw json, got %s", snapshot.RawJSON)
	}
}

func TestStateTelemetryPreservesExistingVersionsWhenPayloadVersionIsEmpty(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:        "drone-version-keep",
		GatewaySn:       "dock-a",
		FirmwareVersion: "05.01.0214",
		HardwareVersion: "M4D",
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewStateTelemetryHandler(db, nil, nil)(ctx, "drone-version-keep", &djisdk.StateMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "", "hardware_version": nil},
	})

	var device struct {
		GatewaySn       string
		FirmwareVersion string
		HardwareVersion string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "drone-version-keep").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	if device.FirmwareVersion != "05.01.0214" || device.HardwareVersion != "M4D" {
		t.Fatalf("device versions = %s/%s, want preserved versions", device.FirmwareVersion, device.HardwareVersion)
	}
}

func TestOsdTelemetryDoesNotUpdateDeviceVersions(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewOsdHandler(db, nil, nil, false)(ctx, "dock-version", &djisdk.OsdMessage{
		Gateway:   "dock-version",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "14.03.00.03", "hardware_version": "Dock3"},
	})

	var device struct {
		FirmwareVersion string
		HardwareVersion string
		DeviceType      string
		DeviceName      string
		IsOnline        bool
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "dock-version").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.FirmwareVersion != "" || device.HardwareVersion != "" {
		t.Fatalf("device versions = %s/%s, want empty because osd must not update state-only versions", device.FirmwareVersion, device.HardwareVersion)
	}
	if device.DeviceType != gormmodel.DjiDeviceUnknown || device.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("device type/name = %q/%q, want unknown sentinels", device.DeviceType, device.DeviceName)
	}
	if !device.IsOnline {
		t.Fatal("expected osd telemetry to mark device online")
	}
}

func TestTelemetryHandlersSkipNilDB(t *testing.T) {
	ctx := context.Background()

	NewOsdHandler(nil, nil, nil, false)(ctx, "dock-nil-db", &djisdk.OsdMessage{
		Gateway:   "dock-nil-db",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "14.03.00.03"},
	})
	NewStateTelemetryHandler(nil, nil, nil)(ctx, "drone-nil-db", &djisdk.StateMessage{
		Gateway:   "dock-nil-db",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "05.01.0214"},
	})
}

func TestStateTelemetryUpdatesGatewayOnEveryReport(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:  "drone-frog-jump",
		GatewaySn: "dock-a",
		IsOnline:  true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewStateTelemetryHandler(db, nil, nil)(ctx, "drone-frog-jump", &djisdk.StateMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"best_link_gateway": "dock-b"},
	})

	var device struct {
		GatewaySn string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn").Where("device_sn = ?", "drone-frog-jump").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
}

func TestStateTelemetryPreservesTopologyAsTypeSource(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:  "payload-1",
		GatewaySn: "dock-a",
		IsOnline:  true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-a",
		SubDeviceSn:   "payload-1",
		Domain:        gormmodel.DjiDeviceDomainPayload,
		SubDeviceType: 99,
	}).Error; err != nil {
		t.Fatalf("create topo error = %v", err)
	}

	NewStateTelemetryHandler(db, nil, nil)(ctx, "payload-1", &djisdk.StateMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"payload_index": "payload-1"},
	})

	var device struct {
		GatewaySn string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn").Where("device_sn = ?", "payload-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	var topo struct {
		Domain string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("domain").Where("gateway_sn = ? AND sub_device_sn = ?", "dock-a", "payload-1").First(&topo).Error; err != nil {
		t.Fatalf("find topo error = %v", err)
	}
	if topo.Domain != gormmodel.DjiDeviceDomainPayload {
		t.Fatalf("topo Domain = %s, want payload domain", topo.Domain)
	}
}

func TestStateTelemetryRejectsMissingGateway(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewStateTelemetryHandler(db, nil, nil)(ctx, "drone-without-gateway", &djisdk.StateMessage{
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1},
	})

	var count int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "drone-without-gateway").Count(&count).Error; err != nil {
		t.Fatalf("count device error = %v", err)
	}
	if count != 0 {
		t.Fatalf("device count = %d, want 0 for invalid state payload", count)
	}
}

func TestStatusUpdateTopoStoresGatewayAndSubDeviceIdentity(t *testing.T) {
	db := newHookTestDB(t)
	onlineCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache online error = %v", err)
	}
	ctx := context.Background()
	msg := &djisdk.StatusMessage{
		Timestamp: 1710000000000,
		Method:    djisdk.MethodUpdateTopo,
		Data: map[string]any{
			"domain":        "3",
			"type":          119,
			"sub_type":      0,
			"device_secret": "secret",
			"thing_version": "1.1.2",
			"sub_devices": []any{
				map[string]any{"sn": "m4d-1", "domain": "0", "type": 60, "sub_type": 0, "index": "A", "device_secret": "secret", "thing_version": "1.1.2"},
			},
		},
	}

	if err := NewStatusHandler(db, onlineCache)(ctx, "dock3-1", msg); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	var dock struct {
		GatewaySn    string
		DeviceType   string
		DeviceName   string
		IsOnline     bool
		LastOnlineAt sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_type", "device_name", "is_online", "last_online_at").Where("device_sn = ?", "dock3-1").First(&dock).Error; err != nil {
		t.Fatalf("find dock error = %v", err)
	}
	if dock.GatewaySn != "dock3-1" {
		t.Fatalf("dock GatewaySn = %s, want dock3-1", dock.GatewaySn)
	}
	if !dock.LastOnlineAt.Valid || dock.LastOnlineAt.Time.IsZero() {
		t.Fatalf("dock LastOnlineAt = %v, want set because status handler now sets online on first create", dock.LastOnlineAt)
	}
	if !dock.IsOnline {
		t.Fatalf("dock IsOnline = false, want true because status handler sets online on first create")
	}
	if dock.DeviceType != "3-119-0" || dock.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("dock device type/name = %q/%q, want 3-119-0/unknown", dock.DeviceType, dock.DeviceName)
	}
	var aircraft struct {
		GatewaySn  string
		DeviceType string
		DeviceName string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_type", "device_name").Where("device_sn = ?", "m4d-1").First(&aircraft).Error; err != nil {
		t.Fatalf("find aircraft error = %v", err)
	}
	if aircraft.GatewaySn != "" {
		t.Fatalf("aircraft GatewaySn = %s, want empty (蛙跳场景 update_topo 不覆盖飞机 gateway_sn)", aircraft.GatewaySn)
	}
	if aircraft.DeviceType != "0-60-0" || aircraft.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("aircraft device type/name = %q/%q, want 0-60-0/Matrice 300 RTK", aircraft.DeviceType, aircraft.DeviceName)
	}
	var topo struct {
		Domain         string
		SubDeviceType  int
		SubDeviceIndex string
		DeviceType     string
		DeviceName     string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("domain", "sub_device_type", "sub_device_index", "device_type", "device_name").Where("gateway_sn = ? AND sub_device_sn = ?", "dock3-1", "m4d-1").First(&topo).Error; err != nil {
		t.Fatalf("find topo error = %v", err)
	}
	if topo.Domain != gormmodel.DjiDeviceDomainAircraft {
		t.Fatalf("topo Domain = %s, want DJI aircraft domain", topo.Domain)
	}
	if topo.SubDeviceType != 60 || topo.SubDeviceIndex != "A" {
		t.Fatalf("topo type/index = %d/%s, want 60/A", topo.SubDeviceType, topo.SubDeviceIndex)
	}
	if topo.DeviceType != "0-60-0" || topo.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("topo device type/name = %q/%q, want 0-60-0/Matrice 300 RTK", topo.DeviceType, topo.DeviceName)
	}
}

func TestStatusUpdateTopoPreservesAircraftAndPayloadGatewayOwnership(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	for _, device := range []gormmodel.DjiDevice{
		{DeviceSn: "aircraft-owned", GatewaySn: "dock-aircraft-current"},
		{DeviceSn: "payload-owned", GatewaySn: "dock-payload-current"},
		{DeviceSn: "remote-owned", GatewaySn: "dock-remote-old"},
	} {
		if err := db.WithContext(ctx).Create(&device).Error; err != nil {
			t.Fatalf("create %s error = %v", device.DeviceSn, err)
		}
	}

	msg := &djisdk.StatusMessage{
		Timestamp: 1710000000000,
		Method:    djisdk.MethodUpdateTopo,
		Data: map[string]any{
			"domain":   "3",
			"type":     3,
			"sub_type": 0,
			"sub_devices": []any{
				map[string]any{"sn": "aircraft-owned", "domain": "0", "type": 60, "sub_type": 0},
				map[string]any{"sn": "payload-owned", "domain": "1", "type": 83, "sub_type": 0},
				map[string]any{"sn": "remote-owned", "domain": "2", "type": 174, "sub_type": 0},
			},
		},
	}
	if err := NewStatusHandler(db, nil)(ctx, "dock-topology", msg); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	tests := []struct {
		deviceSn   string
		gatewaySn  string
		deviceType string
		deviceName string
	}{
		{deviceSn: "aircraft-owned", gatewaySn: "dock-aircraft-current", deviceType: "0-60-0", deviceName: "Matrice 300 RTK"},
		{deviceSn: "payload-owned", gatewaySn: "dock-payload-current", deviceType: "1-83-0", deviceName: "禅思 H30T"},
		{deviceSn: "remote-owned", gatewaySn: "dock-topology", deviceType: "2-174-0", deviceName: "DJI RC Plus 2"},
	}
	for _, tt := range tests {
		var device gormmodel.DjiDevice
		if err := db.WithContext(ctx).Where("device_sn = ?", tt.deviceSn).First(&device).Error; err != nil {
			t.Fatalf("find %s error = %v", tt.deviceSn, err)
		}
		if device.GatewaySn != tt.gatewaySn || device.DeviceType != tt.deviceType || device.DeviceName != tt.deviceName {
			t.Fatalf("%s gateway/type/name = %q/%q/%q, want %q/%q/%q", tt.deviceSn, device.GatewaySn, device.DeviceType, device.DeviceName, tt.gatewaySn, tt.deviceType, tt.deviceName)
		}
	}
}

func TestStatusUpdateTopoStoresUnknownDeviceTypeAndUpdatesDerivedFields(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	handler := NewStatusHandler(db, nil)
	NewStateTelemetryHandler(db, nil, nil)(ctx, "device-derived", &djisdk.StateMessage{
		Gateway:   "dock-current",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "initial"},
	})
	var telemetryDevice gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "device-derived").First(&telemetryDevice).Error; err != nil {
		t.Fatalf("find state-created device error = %v", err)
	}
	if telemetryDevice.DeviceType != gormmodel.DjiDeviceUnknown || telemetryDevice.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("state-created device type/name = %q/%q, want unknown sentinels", telemetryDevice.DeviceType, telemetryDevice.DeviceName)
	}

	message := func(domain string, deviceType, subType int) *djisdk.StatusMessage {
		return &djisdk.StatusMessage{
			Timestamp: 1710000000000,
			Method:    djisdk.MethodUpdateTopo,
			Data: map[string]any{
				"sub_devices": []any{
					map[string]any{"sn": "device-derived", "domain": domain, "type": deviceType, "sub_type": subType},
				},
			},
		}
	}

	if err := handler(ctx, "dock-derived", message("0", 999, 7)); err != nil {
		t.Fatalf("store unknown device type error = %v", err)
	}
	var gateway gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "dock-derived").First(&gateway).Error; err != nil {
		t.Fatalf("find gateway with missing type error = %v", err)
	}
	if gateway.DeviceType != gormmodel.DjiDeviceUnknown || gateway.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("gateway with missing type/name = %q/%q, want unknown sentinels", gateway.DeviceType, gateway.DeviceName)
	}
	var topo gormmodel.DjiDeviceTopo
	if err := db.WithContext(ctx).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-derived", "device-derived").First(&topo).Error; err != nil {
		t.Fatalf("find unknown device type error = %v", err)
	}
	if topo.DeviceType != "0-999-7" || topo.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("unknown device type/name = %q/%q, want 0-999-7/unknown", topo.DeviceType, topo.DeviceName)
	}
	var unknownDevice gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "device-derived").First(&unknownDevice).Error; err != nil {
		t.Fatalf("find unknown device error = %v", err)
	}
	if unknownDevice.DeviceType != "0-999-7" || unknownDevice.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("unknown main device type/name = %q/%q, want 0-999-7/unknown", unknownDevice.DeviceType, unknownDevice.DeviceName)
	}
	if unknownDevice.GatewaySn != "dock-current" {
		t.Fatalf("unknown main device gateway = %q, want state-owned dock-current", unknownDevice.GatewaySn)
	}

	if err := handler(ctx, "dock-derived", message("1", 83, 0)); err != nil {
		t.Fatalf("update known device type error = %v", err)
	}
	if err := db.WithContext(ctx).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-derived", "device-derived").First(&topo).Error; err != nil {
		t.Fatalf("find updated device type error = %v", err)
	}
	if topo.Domain != "1" || topo.SubDeviceType != 83 || topo.SubDeviceSubType != 0 || topo.DeviceType != "1-83-0" || topo.DeviceName != "禅思 H30T" {
		t.Fatalf("updated topo fields = %+v", topo)
	}
	var device gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "device-derived").First(&device).Error; err != nil {
		t.Fatalf("find updated device error = %v", err)
	}
	if device.DeviceType != "1-83-0" || device.DeviceName != "禅思 H30T" {
		t.Fatalf("updated device type/name = %q/%q, want 1-83-0/禅思 H30T", device.DeviceType, device.DeviceName)
	}
}

func TestStatusUpdateTopoInvalidIdentityDoesNotOverwriteKnownValues(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn: "device-invalid", GatewaySn: "dock-current", DeviceType: "0-60-0", DeviceName: "Matrice 300 RTK",
	}).Error; err != nil {
		t.Fatalf("create known device error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn: "dock-invalid", SubDeviceSn: "device-invalid", Domain: "0", DeviceType: "0-60-0", DeviceName: "Matrice 300 RTK",
	}).Error; err != nil {
		t.Fatalf("create known topology error = %v", err)
	}

	msg := &djisdk.StatusMessage{Method: djisdk.MethodUpdateTopo, Data: map[string]any{
		"sub_devices": []any{
			map[string]any{"sn": "device-invalid", "domain": "9", "type": 60, "sub_type": 0},
			map[string]any{"sn": "device-new-invalid", "domain": "9", "type": 60, "sub_type": 0},
		},
	}}
	if err := NewStatusHandler(db, nil)(ctx, "dock-invalid", msg); err != nil {
		t.Fatalf("status handler error = %v", err)
	}

	var existing gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "device-invalid").First(&existing).Error; err != nil {
		t.Fatalf("find existing device error = %v", err)
	}
	if existing.GatewaySn != "dock-invalid" || existing.DeviceType != "0-60-0" || existing.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("existing gateway/type/name = %q/%q/%q, want updated gateway and known identity preserved", existing.GatewaySn, existing.DeviceType, existing.DeviceName)
	}
	var existingTopo gormmodel.DjiDeviceTopo
	if err := db.WithContext(ctx).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-invalid", "device-invalid").First(&existingTopo).Error; err != nil {
		t.Fatalf("find existing topology error = %v", err)
	}
	if existingTopo.DeviceType != "0-60-0" || existingTopo.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("existing topology type/name = %q/%q, want known values preserved", existingTopo.DeviceType, existingTopo.DeviceName)
	}

	var fresh gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "device-new-invalid").First(&fresh).Error; err != nil {
		t.Fatalf("find fresh invalid device error = %v", err)
	}
	if fresh.GatewaySn != "dock-invalid" || fresh.DeviceType != gormmodel.DjiDeviceUnknown || fresh.DeviceName != gormmodel.DjiDeviceUnknown {
		t.Fatalf("fresh invalid gateway/type/name = %q/%q/%q, want dock-invalid/unknown/unknown", fresh.GatewaySn, fresh.DeviceType, fresh.DeviceName)
	}
}

func TestStatusUpdateTopoClearsOnlyMissingSubDevices(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-diff",
		SubDeviceSn:   "old-drone",
		Domain:        gormmodel.DjiDeviceDomainAircraft,
		SubDeviceType: 60,
		ThingVersion:  "old",
	}).Error; err != nil {
		t.Fatalf("create old topo error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-diff",
		SubDeviceSn:   "keep-drone",
		Domain:        gormmodel.DjiDeviceDomainAircraft,
		SubDeviceType: 60,
		ThingVersion:  "old",
	}).Error; err != nil {
		t.Fatalf("create keep topo error = %v", err)
	}

	msg := &djisdk.StatusMessage{
		Timestamp: 1710000000000,
		Method:    djisdk.MethodUpdateTopo,
		Data: map[string]any{
			"domain":        "3",
			"type":          119,
			"sub_type":      0,
			"device_secret": "secret",
			"sub_devices": []any{
				map[string]any{"sn": "keep-drone", "domain": "0", "type": 60, "sub_type": 0, "index": "A", "thing_version": "1.2.3"},
			},
		},
	}

	if err := NewStatusHandler(db, nil)(ctx, "dock-diff", msg); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	var oldCount int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-diff", "old-drone").Count(&oldCount).Error; err != nil {
		t.Fatalf("count old topo error = %v", err)
	}
	if oldCount != 0 {
		t.Fatalf("old topo count = %d, want 0 for missing sub device", oldCount)
	}

	var keep struct {
		SubDeviceIndex string
		ThingVersion   string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("sub_device_index", "thing_version").Where("gateway_sn = ? AND sub_device_sn = ?", "dock-diff", "keep-drone").First(&keep).Error; err != nil {
		t.Fatalf("find keep topo error = %v", err)
	}
	if keep.SubDeviceIndex != "A" || keep.ThingVersion != "1.2.3" {
		t.Fatalf("keep topo index/version = %s/%s, want A/1.2.3", keep.SubDeviceIndex, keep.ThingVersion)
	}
}

func TestStatusUpdateTopoClearsOfflineSubDevices(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-offline",
		SubDeviceSn:   "old-drone",
		Domain:        gormmodel.DjiDeviceDomainAircraft,
		SubDeviceType: 60,
	}).Error; err != nil {
		t.Fatalf("create topo error = %v", err)
	}

	msg := &djisdk.StatusMessage{
		Timestamp: 1710000000000,
		Method:    djisdk.MethodUpdateTopo,
		Data: map[string]any{
			"domain":        "3",
			"type":          119,
			"sub_type":      0,
			"device_secret": "secret",
			"sub_devices":   []any{},
		},
	}

	if err := NewStatusHandler(db, nil)(ctx, "dock-offline", msg); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	var count int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-offline", "old-drone").Count(&count).Error; err != nil {
		t.Fatalf("count topo error = %v", err)
	}
	if count != 0 {
		t.Fatalf("topo count = %d, want 0 for offline sub device", count)
	}
}

func TestStatusUpdateTopoRestoresSoftDeletedSubDevice(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:   "drone-restore",
		GatewaySn:  "dock-current-link",
		DeviceType: "0-999-0",
		DeviceName: "stale device name",
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}
	softDeleted := gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-restore",
		SubDeviceSn:   "drone-restore",
		Domain:        gormmodel.DjiDeviceDomainAircraft,
		SubDeviceType: 60,
		DeviceType:    "0-60-0",
		DeviceName:    "stale name",
		ThingVersion:  "old",
	}
	if err := db.WithContext(ctx).Create(&softDeleted).Error; err != nil {
		t.Fatalf("create topo error = %v", err)
	}
	if err := db.WithContext(ctx).Delete(&softDeleted).Error; err != nil {
		t.Fatalf("soft delete topo error = %v", err)
	}

	msg := &djisdk.StatusMessage{
		Timestamp: 1710000000000,
		Method:    djisdk.MethodUpdateTopo,
		Data: map[string]any{
			"domain":        "3",
			"type":          119,
			"sub_type":      0,
			"device_secret": "secret",
			"sub_devices": []any{
				map[string]any{"sn": "drone-restore", "domain": "0", "type": 60, "sub_type": 0, "index": "A", "thing_version": "1.2.3"},
			},
		},
	}

	if err := NewStatusHandler(db, nil)(ctx, "dock-restore", msg); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	var restored struct {
		SubDeviceIndex string
		ThingVersion   string
		DeviceType     string
		DeviceName     string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("sub_device_index", "thing_version", "device_type", "device_name").Where("gateway_sn = ? AND sub_device_sn = ?", "dock-restore", "drone-restore").First(&restored).Error; err != nil {
		t.Fatalf("find restored topo error = %v", err)
	}
	if restored.SubDeviceIndex != "A" || restored.ThingVersion != "1.2.3" {
		t.Fatalf("restored topo index/version = %s/%s, want A/1.2.3", restored.SubDeviceIndex, restored.ThingVersion)
	}
	if restored.DeviceType != "0-60-0" || restored.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("restored device type/name = %q/%q, want latest registry values", restored.DeviceType, restored.DeviceName)
	}
	var device gormmodel.DjiDevice
	if err := db.WithContext(ctx).Where("device_sn = ?", "drone-restore").First(&device).Error; err != nil {
		t.Fatalf("find restored main device error = %v", err)
	}
	if device.GatewaySn != "dock-current-link" {
		t.Fatalf("restored main device GatewaySn = %q, want existing aircraft owner", device.GatewaySn)
	}
	if device.DeviceType != "0-60-0" || device.DeviceName != "Matrice 300 RTK" {
		t.Fatalf("restored main device type/name = %q/%q, want latest registry values", device.DeviceType, device.DeviceName)
	}
}

func TestOsdTelemetryDoesNotOverwriteFirstOnlineAt(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	firstOnlineAt := time.UnixMilli(1700000000000)
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:      "dock-first-online",
		GatewaySn:     "dock-first-online",
		IsOnline:      true,
		FirstOnlineAt: sql.NullTime{Time: firstOnlineAt, Valid: true},
		LastOnlineAt:  sql.NullTime{Time: firstOnlineAt, Valid: true},
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewOsdHandler(db, nil, nil, false)(ctx, "dock-first-online", &djisdk.OsdMessage{
		Gateway:   "dock-first-online",
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1},
	})

	var device struct {
		FirstOnlineAt sql.NullTime
		LastOnlineAt  sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("first_online_at", "last_online_at").Where("device_sn = ?", "dock-first-online").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if !device.FirstOnlineAt.Valid || !device.FirstOnlineAt.Time.Equal(firstOnlineAt) {
		t.Fatalf("FirstOnlineAt = %v, want original %v", device.FirstOnlineAt, firstOnlineAt)
	}
	if !device.LastOnlineAt.Valid || !device.LastOnlineAt.Time.Equal(time.UnixMilli(1710000000000)) {
		t.Fatalf("LastOnlineAt = %v, want latest report time", device.LastOnlineAt)
	}
}

func TestOsdTelemetryPreservesTopologyAsTypeSource(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:  "payload-osd-1",
		GatewaySn: "dock-a",
		IsOnline:  true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDeviceTopo{
		GatewaySn:     "dock-a",
		SubDeviceSn:   "payload-osd-1",
		Domain:        gormmodel.DjiDeviceDomainPayload,
		SubDeviceType: 99,
	}).Error; err != nil {
		t.Fatalf("create topo error = %v", err)
	}

	NewOsdHandler(db, nil, nil, false)(ctx, "payload-osd-1", &djisdk.OsdMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"payload_index": "payload-osd-1"},
	})

	var device struct {
		GatewaySn string
		IsOnline  bool
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "is_online").Where("device_sn = ?", "payload-osd-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	var topo struct {
		Domain string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("domain").Where("gateway_sn = ? AND sub_device_sn = ?", "dock-a", "payload-osd-1").First(&topo).Error; err != nil {
		t.Fatalf("find topo error = %v", err)
	}
	if topo.Domain != gormmodel.DjiDeviceDomainPayload {
		t.Fatalf("topo Domain = %s, want payload domain", topo.Domain)
	}
	if !device.IsOnline {
		t.Fatal("expected osd telemetry to mark payload online")
	}
}

func TestOsdTelemetryRejectsMissingGateway(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewOsdHandler(db, nil, nil, false)(ctx, "osd-without-gateway", &djisdk.OsdMessage{
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1},
	})

	var count int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "osd-without-gateway").Count(&count).Error; err != nil {
		t.Fatalf("count device error = %v", err)
	}
	if count != 0 {
		t.Fatalf("device count = %d, want 0 for invalid osd payload", count)
	}
}

func TestHookHandlersDoNotGenerateDialectUpsertSQL(t *testing.T) {
	db := newHookTestDB(t)
	sqls := make([]string, 0)
	db.DB.Callback().Create().Before("gorm:create").Register("hooks:test_capture_create_sql", func(tx *gorm.DB) {
		tx.Statement.SQL.Reset()
		tx.Statement.Build("INSERT", "VALUES", "ON CONFLICT")
		sqls = append(sqls, tx.Statement.SQL.String())
	})

	ctx := context.Background()
	NewOsdHandler(db, nil, nil, false)(ctx, "dock-sql", &djisdk.OsdMessage{
		Gateway:   "dock-sql",
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1},
	})
	NewStateTelemetryHandler(db, nil, nil)(ctx, "drone-sql", &djisdk.StateMessage{
		Gateway:   "dock-sql",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "05.01.0214"},
	})
	NewFlightTaskProgressHandler(db)(ctx, "dock-sql", &djisdk.FlightTaskProgressEvent{
		Status: "running",
		Progress: djisdk.FlightTaskProgressProgress{
			CurrentStep: 2,
			Percent:     50,
		},
		Ext: djisdk.FlightTaskProgressExt{
			FlightID:             "flight-sql",
			CurrentWaypointIndex: 1,
			WaylineMissionState:  6,
			MediaCount:           3,
			TrackID:              "track-sql",
			WaylineID:            9,
		},
	})
	if err := NewStatusHandler(db, nil)(ctx, "dock-sql", &djisdk.StatusMessage{
		Method:    djisdk.MethodUpdateTopo,
		Timestamp: 1710000000000,
		Data: map[string]any{
			"sub_devices": []any{
				map[string]any{"sn": "drone-sql", "domain": "0", "type": 60, "sub_type": 0, "index": "A"},
			},
		},
	}); err != nil {
		t.Fatalf("status handler error = %v, want nil", err)
	}

	for _, sql := range sqls {
		if strings.Contains(sql, "ON CONFLICT") {
			t.Fatalf("generated dialect upsert SQL %q", sql)
		}
	}
}

func TestOsdTelemetryStoresOnlyOfficialRawSnapshot(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewOsdHandler(db, nil, nil, false)(ctx, "dock-json", &djisdk.OsdMessage{
		Gateway:   "dock-json",
		Timestamp: 1710000000000,
		Data:      map[string]any{"mode_code": 1, "latitude": 22.1},
	})

	var snapshot struct {
		RawJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceOsdSnapshot{}).Select("raw_json").Where("device_sn = ?", "dock-json").First(&snapshot).Error; err != nil {
		t.Fatalf("find osd snapshot error = %v", err)
	}
	if snapshot.RawJSON == "" || snapshot.RawJSON == "{}" {
		t.Fatalf("RawJSON = %q, want raw osd payload", snapshot.RawJSON)
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiDeviceOsdSnapshot{}, "latitude") {
		t.Fatal("expected osd snapshot not to have guessed latitude column")
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiDeviceOsdSnapshot{}, "mode_code") {
		t.Fatal("expected osd snapshot not to have guessed mode_code column")
	}
}

func TestStateTelemetryStoresOnlyOfficialRawSnapshot(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewStateTelemetryHandler(db, nil, nil)(ctx, "dock-state-json", &djisdk.StateMessage{
		Gateway:   "dock-state-json",
		Timestamp: 1710000000000,
		Data:      map[string]any{"wireless_link_topo": map[string]any{"quality": 1}},
	})

	var snapshot struct {
		RawJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceStateSnapshot{}).Select("raw_json").Where("device_sn = ?", "dock-state-json").First(&snapshot).Error; err != nil {
		t.Fatalf("find state snapshot error = %v", err)
	}
	if snapshot.RawJSON == "" || snapshot.RawJSON == "{}" {
		t.Fatalf("RawJSON = %q, want raw state payload", snapshot.RawJSON)
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiDeviceStateSnapshot{}, "sub_device_sn") {
		t.Fatal("expected state snapshot not to have guessed sub_device_sn column")
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiDeviceStateSnapshot{}, "sub_device_online") {
		t.Fatal("expected state snapshot not to have guessed sub_device_online column")
	}
}

func TestFlightTaskProgressStoresOfficialFieldsAndUpdatesDockTaskAndDeviceState(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	data := &djisdk.FlightTaskProgressEvent{
		Ext: djisdk.FlightTaskProgressExt{
			FlightID:             "flight-json",
			WaylineMissionState:  6,
			CurrentWaypointIndex: 3,
			MediaCount:           4,
			TrackID:              "track-json",
			WaylineID:            2,
			BreakPoint: &djisdk.FlightTaskBreakPoint{
				Index: 1,
				State: 2,
			},
		},
		Progress: djisdk.FlightTaskProgressProgress{
			CurrentStep: 2,
			Percent:     70.5,
		},
		Status: "in_progress",
	}

	NewFlightTaskProgressHandler(db)(ctx, "dock-progress", data)

	var task struct {
		Status         string
		CurrentStep    int
		TrackId        string
		WaylineId      int
		RawJSON        string
		BreakPointJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockFlightTask{}).Select("status", "current_step", "track_id", "wayline_id", "raw_json", "break_point_json").Where("gateway_sn = ? AND flight_id = ?", "dock-progress", "flight-json").First(&task).Error; err != nil {
		t.Fatalf("find dock flight task error = %v", err)
	}
	if task.Status != "in_progress" || task.CurrentStep != 2 || task.TrackId != "track-json" || task.WaylineId != 2 {
		t.Fatalf("task official fields = status:%s step:%d track:%s wayline:%d", task.Status, task.CurrentStep, task.TrackId, task.WaylineId)
	}
	if task.RawJSON == "" || task.RawJSON == "{}" {
		t.Fatalf("RawJSON = %q, want raw event data", task.RawJSON)
	}
	if task.BreakPointJSON == "" || task.BreakPointJSON == "{}" {
		t.Fatalf("BreakPointJSON = %q, want raw break point data", task.BreakPointJSON)
	}

	data.Progress.Percent = 88.8
	data.Status = "updated"
	NewFlightTaskProgressHandler(db)(ctx, "dock-progress", data)

	var taskTotal int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockFlightTask{}).Where("gateway_sn = ? AND flight_id = ?", "dock-progress", "flight-json").Count(&taskTotal).Error; err != nil {
		t.Fatalf("count dock flight task error = %v", err)
	}
	if taskTotal != 1 {
		t.Fatalf("dock task count = %d, want 1", taskTotal)
	}
	var snapshot struct {
		Status          string
		ProgressPercent float64
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockFlightTask{}).Select("status", "progress_percent").Where("gateway_sn = ? AND flight_id = ?", "dock-progress", "flight-json").First(&snapshot).Error; err != nil {
		t.Fatalf("find dock flight task error = %v", err)
	}
	if snapshot.Status != "updated" || snapshot.ProgressPercent != 88.8 {
		t.Fatalf("task latest status/percent = %s/%f, want updated/88.8", snapshot.Status, snapshot.ProgressPercent)
	}

	other := *data
	other.Ext.FlightID = "flight-other"
	other.Status = "other-task"
	other.Progress.Percent = 11.1
	NewFlightTaskProgressHandler(db)(ctx, "dock-progress", &other)

	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockFlightTask{}).Select("status", "progress_percent").Where("gateway_sn = ? AND flight_id = ?", "dock-progress", "flight-json").First(&snapshot).Error; err != nil {
		t.Fatalf("find original dock flight task error = %v", err)
	}
	if snapshot.Status != "updated" || snapshot.ProgressPercent != 88.8 {
		t.Fatalf("task latest after other task = %s/%f, want updated/88.8", snapshot.Status, snapshot.ProgressPercent)
	}

	var dockLatest struct {
		FlightId        string
		Status          string
		ProgressPercent float64
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockDeviceFlightTaskState{}).Select("flight_id", "status", "progress_percent").Where("gateway_sn = ?", "dock-progress").First(&dockLatest).Error; err != nil {
		t.Fatalf("find dock device flight task state error = %v", err)
	}
	if dockLatest.FlightId != "flight-other" || dockLatest.Status != "other-task" || dockLatest.ProgressPercent != 11.1 {
		t.Fatalf("dock latest = %s/%s/%f, want flight-other/other-task/11.1", dockLatest.FlightId, dockLatest.Status, dockLatest.ProgressPercent)
	}
}

func TestFlightTaskProgressSkipsInvalidIdentity(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	data := &djisdk.FlightTaskProgressEvent{
		Ext:      djisdk.FlightTaskProgressExt{FlightID: "flight-invalid"},
		Progress: djisdk.FlightTaskProgressProgress{Percent: 10},
		Status:   "in_progress",
	}

	NewFlightTaskProgressHandler(db)(ctx, "", data)
	data.Ext.FlightID = ""
	NewFlightTaskProgressHandler(db)(ctx, "dock-invalid", data)

	var taskTotal int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockFlightTask{}).Count(&taskTotal).Error; err != nil {
		t.Fatalf("count dock flight task error = %v", err)
	}
	if taskTotal != 0 {
		t.Fatalf("dock task count = %d, want 0", taskTotal)
	}
	var stateTotal int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDockDeviceFlightTaskState{}).Count(&stateTotal).Error; err != nil {
		t.Fatalf("count dock device flight task state error = %v", err)
	}
	if stateTotal != 0 {
		t.Fatalf("dock device state count = %d, want 0", stateTotal)
	}
}

func TestHmsAlertStoresOfficialItemJSON(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	resolver := djisdk.MustNewHmsResolver(djisdk.HmsConfig{})
	NewHmsEventNotifyHandler(db, resolver)(ctx, "dock-hms", &djisdk.HmsEventData{List: []djisdk.HmsItem{{
		Level:      2,
		Module:     3,
		InTheSky:   1,
		Code:       "0x16100083",
		DeviceType: "0-67-0",
		Imminent:   1,
		Args: map[string]any{
			"component_index": 4, "sensor_index": 2, "alarmid": "0x16100001",
			"gimbal_index": 6, "lidar_index": 7, "lte_index": 8, "future_arg": "kept",
		},
	}}})

	var alert struct {
		ItemJSON       string
		Message        string
		MessageKey     string
		DeviceDomain   int
		DeviceTypeID   int
		DeviceSubtype  int
		DeviceTypeName string
		AlarmID        string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiHmsAlert{}).Where("gateway_sn = ?", "dock-hms").First(&alert).Error; err != nil {
		t.Fatalf("find hms alert error = %v", err)
	}
	if alert.ItemJSON == "" || alert.ItemJSON == "{}" {
		t.Fatalf("ItemJSON = %q, want raw hms item", alert.ItemJSON)
	}
	if alert.Message == "" {
		t.Fatal("Message is empty, want resolved HMS message")
	}
	if !strings.Contains(alert.ItemJSON, `"future_arg":"kept"`) {
		t.Fatalf("ItemJSON = %q, want unknown args preserved", alert.ItemJSON)
	}
	if alert.DeviceDomain != 0 || alert.DeviceTypeID != 67 || alert.DeviceSubtype != 0 || alert.DeviceTypeName != "Matrice 30" || alert.AlarmID != "0x16100001" || alert.MessageKey != "fpv_tip_0x16100083_in_the_sky" {
		t.Fatalf("flattened HMS fields = %+v", alert)
	}
	for _, column := range []string{"component_index", "sensor_index", "gimbal_index", "lidar_index", "lte_index"} {
		if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiHmsAlert{}, column) {
			t.Fatalf("expected HMS alert not to have flattened %s column", column)
		}
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiHmsAlert{}, "device_sn") {
		t.Fatal("expected hms alert not to have guessed device_sn column")
	}
	if !db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiHmsAlert{}, "message") {
		t.Fatal("expected hms alert to have message column")
	}
}

func TestHmsAlertStoresCorrelationTraceAndItemSnapshots(t *testing.T) {
	db := newHookTestDB(t)
	dictionaryPath := filepath.Join(t.TempDir(), "hms.json")
	if err := os.WriteFile(dictionaryPath, []byte(`{
  "fpv_tip_exact_in_the_sky": {"zh": "空中命中文案"},
  "fpv_tip_fallback": {"zh": "空中回退文案，云台索引 %gimbal_index"}
}`), 0o600); err != nil {
		t.Fatalf("write HMS dictionary error = %v", err)
	}

	traceID := gooteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	ctx := gooteltrace.ContextWithRemoteSpanContext(context.Background(), gooteltrace.NewSpanContext(gooteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  gooteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Remote:  true,
	}))
	resolver := djisdk.MustNewHmsResolver(djisdk.HmsConfig{DictionaryPath: dictionaryPath})
	client := djisdk.NewClient(nil, djisdk.WithHmsEventNotifyHandler(NewHmsEventNotifyHandler(db, resolver)))
	payload := []byte(`{"tid":"tid-placement","bid":"bid-placement","gateway":"dock-placement","method":"hms","data":{"list":[{"code":"exact","device_type":"0-67-0","in_the_sky":1},{"code":"fallback","device_type":"0-67-0","in_the_sky":1,"args":{"gimbal_index":7}},{"code":"unknown","device_type":"1-83-0","args":{"gimbal_index":0}}]}}`)
	if err := client.HandleEvents(ctx, payload, djisdk.EventsTopic("dock-placement"), ""); err != nil {
		t.Fatalf("handle HMS event error = %v", err)
	}

	var alerts []gormmodel.DjiHmsAlert
	if err := db.Where("gateway_sn = ?", "dock-placement").Find(&alerts).Error; err != nil {
		t.Fatalf("find HMS alerts error = %v", err)
	}
	if len(alerts) != 3 {
		t.Fatalf("HMS alert count = %d, want 3", len(alerts))
	}
	wantSnapshots := map[string]struct {
		messageKey     string
		message        string
		deviceTypeName string
	}{
		"exact":    {messageKey: "fpv_tip_exact_in_the_sky", message: "空中命中文案", deviceTypeName: "Matrice 30"},
		"fallback": {messageKey: "fpv_tip_fallback", message: "空中回退文案，云台索引 7", deviceTypeName: "Matrice 30"},
		"unknown":  {message: "未知 HMS 告警（unknown）", deviceTypeName: "禅思 H30T"},
	}
	for _, alert := range alerts {
		if alert.Tid != "tid-placement" || alert.Bid != "bid-placement" || alert.TraceID != traceID.String() {
			t.Fatalf("HMS correlation = %q/%q/%q", alert.Tid, alert.Bid, alert.TraceID)
		}
		want, ok := wantSnapshots[alert.Code]
		if !ok {
			t.Fatalf("unexpected HMS code %q", alert.Code)
		}
		if alert.DeviceTypeName != want.deviceTypeName || alert.MessageKey != want.messageKey || alert.Message != want.message || alert.ItemJSON == "" {
			t.Fatalf("HMS snapshot fields = %+v", alert)
		}
	}
	if alerts[0].ItemJSON == alerts[1].ItemJSON || alerts[1].ItemJSON == alerts[2].ItemJSON {
		t.Fatalf("HMS items were not persisted as distinct history rows: %+v", alerts)
	}
}

func TestHmsAlertLeavesUnknownDeviceTypeNameEmpty(t *testing.T) {
	db := newHookTestDB(t)
	resolver := djisdk.MustNewHmsResolver(djisdk.HmsConfig{})
	NewHmsEventNotifyHandler(db, resolver)(context.Background(), "dock-hms-unknown", &djisdk.HmsEventData{List: []djisdk.HmsItem{{
		Code:       "0xDEADBEEF",
		DeviceType: "0-999-0",
	}}})

	var alert gormmodel.DjiHmsAlert
	if err := db.Where("gateway_sn = ?", "dock-hms-unknown").First(&alert).Error; err != nil {
		t.Fatalf("find hms alert error = %v", err)
	}
	if alert.DeviceTypeName != "" || alert.MessageKey != "" || alert.DeviceDomain != 0 || alert.DeviceTypeID != 999 || alert.DeviceSubtype != 0 {
		t.Fatalf("unknown device fields = %+v", alert)
	}
}

func TestReturnHomeEventDoesNotStoreLeapfrogDerivedFields(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewReturnHomeInfoHandler(db)(ctx, "dock-return", &djisdk.ReturnHomeInfoEvent{
		FlightID:      "flight-return",
		HomeDockSn:    "dock-home",
		LastPointType: 1,
		PlannedPathPoints: []djisdk.PathPoint{{
			Latitude:  22.1,
			Longitude: 113.1,
		}},
		MultiDockHomeInfo: []djisdk.DockHomeInfo{{
			SN:           "dock-a",
			HomeDistance: 12.3,
		}},
	})

	var event struct {
		FlightId              string
		PlannedPathPointCount int
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiReturnHomeEvent{}).Select("flight_id", "planned_path_point_count").Where("gateway_sn = ?", "dock-return").First(&event).Error; err != nil {
		t.Fatalf("find return home event error = %v", err)
	}
	if event.FlightId != "flight-return" || event.PlannedPathPointCount != 1 {
		t.Fatalf("return home event = %s/%d, want flight-return/1", event.FlightId, event.PlannedPathPointCount)
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiReturnHomeEvent{}, "multi_dock_home_info_count") {
		t.Fatal("expected return home event not to have leapfrog multi_dock_home_info_count column")
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiReturnHomeEvent{}, "nearest_home_distance") {
		t.Fatal("expected return home event not to have leapfrog nearest_home_distance column")
	}
}

func TestIsOnlineWithNilCacheReturnsFalse(t *testing.T) {
	if IsOnline(nil, "gateway-1") {
		t.Fatal("expected nil online cache to report offline")
	}
}

func TestFlightTaskReadyPersistsEventWithFlightIDs(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewFlightTaskReadyHandler(db)(ctx, "dock-ready", &djisdk.FlightTaskReadyEvent{
		FlightIDs: []string{"flight-a", "flight-b"},
	})

	var ready struct {
		GatewaySn   string
		FlightCount int
		RawJSON     string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiFlightTaskReady{}).Select("gateway_sn", "flight_count", "raw_json").Where("gateway_sn = ?", "dock-ready").First(&ready).Error; err != nil {
		t.Fatalf("find flight task ready error = %v", err)
	}
	if ready.GatewaySn != "dock-ready" {
		t.Fatalf("GatewaySn = %s, want dock-ready", ready.GatewaySn)
	}
	if ready.FlightCount != 2 {
		t.Fatalf("FlightCount = %d, want 2", ready.FlightCount)
	}
	if ready.RawJSON == "" || ready.RawJSON == "{}" {
		t.Fatalf("RawJSON = %q, want raw event data", ready.RawJSON)
	}
}

func TestRemoteLogProgressPersistsEventWithMethod(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewRemoteLogFileUploadProgressHandler(db)(ctx, "dock-log-p", &djisdk.RemoteLogFileUploadProgressEvent{
		Files: []djisdk.RemoteLogFileUploadProgress{
			{DeviceSN: "dock-log-p", Module: "dock", Key: "log-1", Progress: 50},
		},
	})

	var event struct {
		Method    string
		FileCount int
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiRemoteLogEvent{}).Select("method", "file_count").Where("gateway_sn = ?", "dock-log-p").First(&event).Error; err != nil {
		t.Fatalf("find remote log progress event error = %v", err)
	}
	if event.Method != "fileupload_progress" {
		t.Fatalf("Method = %s, want fileupload_progress", event.Method)
	}
	if event.FileCount != 1 {
		t.Fatalf("FileCount = %d, want 1", event.FileCount)
	}
}

func TestDeviceRequestHandlerReturnsErrorOnNilReq(t *testing.T) {
	handler := NewDeviceRequestHandler(nil, nil)
	_, err := handler(context.Background(), "dock-nil", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestDrcUpPersistsStickControlEvent(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	err := NewDrcUpHandler(db, nil)(ctx, "dock-drc", &djisdk.DrcUpMessage{Method: djisdk.MethodStickControl, Timestamp: 1710000000000}, &djisdk.DrcStickControlAckData{Result: 0})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}

	var event struct {
		RawJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDrcUpEvent{}).Select("raw_json").Where("gateway_sn = ? AND method = ?", "dock-drc", djisdk.MethodStickControl).First(&event).Error; err != nil {
		t.Fatalf("find drc event error = %v", err)
	}
	if event.RawJSON != `{"result":0}` {
		t.Fatalf("RawJSON = %q, want stick_control ack", event.RawJSON)
	}
}

func TestDrcUpHandlerIgnoresNilMessage(t *testing.T) {
	if err := NewDrcUpHandler(nil, nil)(context.Background(), "dock-nil", nil, nil); err != nil {
		t.Fatalf("handler error = %v", err)
	}
}

func TestDeviceRequestHandlerReturnsMethodSpecificOutput(t *testing.T) {
	handler := NewDeviceRequestHandler(nil, nil)
	cases := []struct {
		method string
		want   string
	}{
		{method: djisdk.MethodFlightAreasGet, want: "files"},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			output, err := handler(context.Background(), "dock-req", &djisdk.RequestMessage{Method: tc.method})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if output == nil || !strings.Contains(asJSON(t, output), tc.want) {
				t.Fatalf("output = %#v, want key %s", output, tc.want)
			}
		})
	}
}

func TestDeviceRequestHandlerSkipsReplyForLegacyMethods(t *testing.T) {
	handler := NewDeviceRequestHandler(nil, nil)
	methods := []string{djisdk.MethodAirportOrganizationGet, djisdk.MethodAirportBindStatus}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			_, err := handler(context.Background(), "dock-req", &djisdk.RequestMessage{Method: method})
			if !errors.Is(err, djisdk.ErrSkipRequestReply) {
				t.Fatalf("handler for %s should return ErrSkipRequestReply, got %v", method, err)
			}
		})
	}
}

func asJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal output error = %v", err)
	}
	return string(data)
}
