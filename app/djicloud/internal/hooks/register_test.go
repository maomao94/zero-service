package hooks

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
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
	client := djisdk.NewClient(nil, djisdk.WithPendingTTL(time.Second), djisdk.WithReplyOptions(djisdk.ReplyOptions{}))

	RegisterDjiClient(client, RegisterDjiClientOptions{
		DB:          newHookTestDB(t),
		OnlineCache: onlineCache,
	})

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
	} else if err.Error() != "[dji-sdk] device offline: sn=offline-gateway, command rejected" {
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
}

func TestRegisterDjiClientWithoutOnlineCacheHandlesUpstreamWithoutOnlineChecker(t *testing.T) {
	client := djisdk.NewClient(nil, djisdk.WithPendingTTL(time.Second), djisdk.WithReplyOptions(djisdk.ReplyOptions{}))

	RegisterDjiClient(client, RegisterDjiClientOptions{})

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

	NewStateTelemetryHandler(db, onlineCache)(ctx, "drone-1", &djisdk.StateMessage{
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

	NewStateTelemetryHandler(db, nil)(ctx, "drone-version-keep", &djisdk.StateMessage{
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

	NewOsdHandler(db, nil)(ctx, "dock-version", &djisdk.OsdMessage{
		Gateway:   "dock-version",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "14.03.00.03", "hardware_version": "Dock3"},
	})

	var device struct {
		FirmwareVersion string
		HardwareVersion string
		IsOnline        bool
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Where("device_sn = ?", "dock-version").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.FirmwareVersion != "" || device.HardwareVersion != "" {
		t.Fatalf("device versions = %s/%s, want empty because osd must not update state-only versions", device.FirmwareVersion, device.HardwareVersion)
	}
	if !device.IsOnline {
		t.Fatal("expected osd telemetry to mark device online")
	}
}

func TestTelemetryHandlersSkipNilDB(t *testing.T) {
	ctx := context.Background()

	NewOsdHandler(nil, nil)(ctx, "dock-nil-db", &djisdk.OsdMessage{
		Gateway:   "dock-nil-db",
		Timestamp: 1710000000000,
		Data:      map[string]any{"firmware_version": "14.03.00.03"},
	})
	NewStateTelemetryHandler(nil, nil)(ctx, "drone-nil-db", &djisdk.StateMessage{
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

	NewStateTelemetryHandler(db, nil)(ctx, "drone-frog-jump", &djisdk.StateMessage{
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

	NewStateTelemetryHandler(db, nil)(ctx, "payload-1", &djisdk.StateMessage{
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

	NewStateTelemetryHandler(db, nil)(ctx, "drone-without-gateway", &djisdk.StateMessage{
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

func TestStatusUpdateTopoStoresTypeOnlyInTopo(t *testing.T) {
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

	result := NewStatusHandler(db, onlineCache)(ctx, "dock3-1", msg)
	if result != djisdk.PlatformResultOK {
		t.Fatalf("status result = %d, want %d", result, djisdk.PlatformResultOK)
	}

	var dock struct {
		GatewaySn    string
		LastOnlineAt sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "last_online_at").Where("device_sn = ?", "dock3-1").First(&dock).Error; err != nil {
		t.Fatalf("find dock error = %v", err)
	}
	if dock.GatewaySn != "dock3-1" {
		t.Fatalf("dock GatewaySn = %s, want dock3-1", dock.GatewaySn)
	}
	if dock.LastOnlineAt.Valid {
		t.Fatalf("dock LastOnlineAt = %v, want empty because status does not handle online", dock.LastOnlineAt)
	}
	var aircraft struct {
		GatewaySn string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn").Where("device_sn = ?", "m4d-1").First(&aircraft).Error; err != nil {
		t.Fatalf("find aircraft error = %v", err)
	}
	if aircraft.GatewaySn != "dock3-1" {
		t.Fatalf("aircraft GatewaySn = %s, want dock3-1", aircraft.GatewaySn)
	}
	var topo struct {
		Domain         string
		SubDeviceType  int
		SubDeviceIndex string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Select("domain", "sub_device_type", "sub_device_index").Where("gateway_sn = ? AND sub_device_sn = ?", "dock3-1", "m4d-1").First(&topo).Error; err != nil {
		t.Fatalf("find topo error = %v", err)
	}
	if topo.Domain != gormmodel.DjiDeviceDomainAircraft {
		t.Fatalf("topo Domain = %s, want DJI aircraft domain", topo.Domain)
	}
	if topo.SubDeviceType != 60 || topo.SubDeviceIndex != "A" {
		t.Fatalf("topo type/index = %d/%s, want 60/A", topo.SubDeviceType, topo.SubDeviceIndex)
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

	result := NewStatusHandler(db, nil)(ctx, "dock-diff", msg)
	if result != djisdk.PlatformResultOK {
		t.Fatalf("status result = %d, want %d", result, djisdk.PlatformResultOK)
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

	result := NewStatusHandler(db, nil)(ctx, "dock-offline", msg)
	if result != djisdk.PlatformResultOK {
		t.Fatalf("status result = %d, want %d", result, djisdk.PlatformResultOK)
	}

	var count int64
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDeviceTopo{}).Where("gateway_sn = ? AND sub_device_sn = ?", "dock-offline", "old-drone").Count(&count).Error; err != nil {
		t.Fatalf("count topo error = %v", err)
	}
	if count != 0 {
		t.Fatalf("topo count = %d, want 0 for offline sub device", count)
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

	NewOsdHandler(db, nil)(ctx, "dock-first-online", &djisdk.OsdMessage{
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

	NewOsdHandler(db, nil)(ctx, "payload-osd-1", &djisdk.OsdMessage{
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

	NewOsdHandler(db, nil)(ctx, "osd-without-gateway", &djisdk.OsdMessage{
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

func TestOsdTelemetryStoresOnlyOfficialRawSnapshot(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()

	NewOsdHandler(db, nil)(ctx, "dock-json", &djisdk.OsdMessage{
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

	NewStateTelemetryHandler(db, nil)(ctx, "dock-state-json", &djisdk.StateMessage{
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

	NewHmsEventNotifyHandler(db)(ctx, "dock-hms", &djisdk.HmsEventData{List: []djisdk.HmsItem{{
		Level:      2,
		Module:     3,
		InTheSky:   1,
		Code:       "0x16100083",
		DeviceType: "dock",
		Imminent:   1,
		Args:       djisdk.HmsItemArgs{ComponentIndex: 4, SensorIndex: 5},
	}}})

	var alert struct {
		ItemJSON string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiHmsAlert{}).Select("item_json").Where("gateway_sn = ?", "dock-hms").First(&alert).Error; err != nil {
		t.Fatalf("find hms alert error = %v", err)
	}
	if alert.ItemJSON == "" || alert.ItemJSON == "{}" {
		t.Fatalf("ItemJSON = %q, want raw hms item", alert.ItemJSON)
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiHmsAlert{}, "device_sn") {
		t.Fatal("expected hms alert not to have guessed device_sn column")
	}
	if db.WithContext(ctx).Migrator().HasColumn(&gormmodel.DjiHmsAlert{}, "message") {
		t.Fatal("expected hms alert not to have guessed message column")
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
	handler := NewDeviceRequestHandler()
	result, _, _ := handler(context.Background(), "dock-nil", nil)
	if result != djisdk.PlatformResultHandlerError {
		t.Fatalf("result = %d, want PlatformResultHandlerError for nil request", result)
	}
}
