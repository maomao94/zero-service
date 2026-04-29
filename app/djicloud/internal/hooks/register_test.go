package hooks

import (
	"context"
	"database/sql"
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

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&parseTime=true&loc=UTC"), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Unix(1710000000, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("open sqlite db error = %v", err)
	}
	if err := db.AutoMigrate(&gormmodel.DjiDevice{}, &gormmodel.DjiDeviceTopo{}, &gormmodel.DjiDeviceOsdSnapshot{}, &gormmodel.DjiDeviceStateSnapshot{}); err != nil {
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
		Data:      map[string]any{"mode_code": 1},
	})

	if IsOnline(onlineCache, "drone-1") {
		t.Fatal("expected state telemetry not to refresh device online cache")
	}
	if IsOnline(onlineCache, "dock-1") {
		t.Fatal("expected state telemetry not to refresh gateway online cache")
	}
	var device struct {
		GatewaySn    string
		DeviceDomain string
		IsOnline     bool
		LastOnlineAt sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_domain", "is_online", "last_online_at").Where("device_sn = ?", "drone-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-1" {
		t.Fatalf("GatewaySn = %s, want dock-1", device.GatewaySn)
	}
	if device.DeviceDomain != "" {
		t.Fatalf("DeviceDomain = %s, want zero value because state does not carry DJI domain", device.DeviceDomain)
	}
	if device.IsOnline {
		t.Fatal("expected state telemetry not to mark device online")
	}
	if device.LastOnlineAt.Valid {
		t.Fatal("expected state telemetry not to update last online time")
	}
}

func TestStateTelemetryUpdatesGatewayOnEveryReport(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:     "drone-frog-jump",
		GatewaySn:    "dock-a",
		DeviceDomain: gormmodel.DjiDeviceDomainAircraft,
		IsOnline:     true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewStateTelemetryHandler(db, nil)(ctx, "drone-frog-jump", &djisdk.StateMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"best_link_gateway": "dock-b"},
	})

	var device struct {
		GatewaySn    string
		DeviceDomain string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_domain").Where("device_sn = ?", "drone-frog-jump").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	if device.DeviceDomain != gormmodel.DjiDeviceDomainAircraft {
		t.Fatalf("DeviceDomain = %s, want existing aircraft domain", device.DeviceDomain)
	}
}

func TestStateTelemetryPreservesKnownPayloadDomain(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:     "payload-1",
		GatewaySn:    "dock-a",
		DeviceDomain: gormmodel.DjiDeviceDomainPayload,
		IsOnline:     true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewStateTelemetryHandler(db, nil)(ctx, "payload-1", &djisdk.StateMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"payload_index": "payload-1"},
	})

	var device struct {
		GatewaySn    string
		DeviceDomain string
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_domain").Where("device_sn = ?", "payload-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	if device.DeviceDomain != gormmodel.DjiDeviceDomainPayload {
		t.Fatalf("DeviceDomain = %s, want existing payload domain because state does not carry DJI domain", device.DeviceDomain)
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

func TestStatusUpdateTopoUsesDjiDomains(t *testing.T) {
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
		DeviceDomain  string
		DeviceType    int
		DeviceSubType int
		LastOnlineAt  sql.NullTime
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("device_domain", "device_type", "device_sub_type", "last_online_at").Where("device_sn = ?", "dock3-1").First(&dock).Error; err != nil {
		t.Fatalf("find dock error = %v", err)
	}
	if dock.DeviceDomain != gormmodel.DjiDeviceDomainDock {
		t.Fatalf("dock DeviceDomain = %s, want DJI dock domain", dock.DeviceDomain)
	}
	if dock.DeviceType != 119 || dock.DeviceSubType != 0 {
		t.Fatalf("dock type = %d/%d, want 119/0", dock.DeviceType, dock.DeviceSubType)
	}
	if dock.LastOnlineAt.Valid {
		t.Fatalf("dock LastOnlineAt = %v, want empty because status does not handle online", dock.LastOnlineAt)
	}
	var aircraft struct {
		DeviceDomain string
		DeviceType   int
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("device_domain", "device_type").Where("device_sn = ?", "m4d-1").First(&aircraft).Error; err != nil {
		t.Fatalf("find aircraft error = %v", err)
	}
	if aircraft.DeviceDomain != gormmodel.DjiDeviceDomainAircraft {
		t.Fatalf("aircraft DeviceDomain = %s, want DJI aircraft domain", aircraft.DeviceDomain)
	}
	if aircraft.DeviceType != 60 {
		t.Fatalf("aircraft DeviceType = %d, want 60", aircraft.DeviceType)
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
		DeviceDomain:  gormmodel.DjiDeviceDomainDock,
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

func TestOsdTelemetryPreservesKnownPayloadDomain(t *testing.T) {
	db := newHookTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&gormmodel.DjiDevice{
		DeviceSn:     "payload-osd-1",
		GatewaySn:    "dock-a",
		DeviceDomain: gormmodel.DjiDeviceDomainPayload,
		IsOnline:     true,
	}).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	NewOsdHandler(db, nil)(ctx, "payload-osd-1", &djisdk.OsdMessage{
		Gateway:   "dock-b",
		Timestamp: 1710000000000,
		Data:      map[string]any{"payload_index": "payload-osd-1"},
	})

	var device struct {
		GatewaySn    string
		DeviceDomain string
		IsOnline     bool
	}
	if err := db.WithContext(ctx).Model(&gormmodel.DjiDevice{}).Select("gateway_sn", "device_domain", "is_online").Where("device_sn = ?", "payload-osd-1").First(&device).Error; err != nil {
		t.Fatalf("find device error = %v", err)
	}
	if device.GatewaySn != "dock-b" {
		t.Fatalf("GatewaySn = %s, want dock-b", device.GatewaySn)
	}
	if device.DeviceDomain != gormmodel.DjiDeviceDomainPayload {
		t.Fatalf("DeviceDomain = %s, want existing payload domain because osd does not carry DJI domain", device.DeviceDomain)
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

func TestIsOnlineWithNilCacheReturnsFalse(t *testing.T) {
	if IsOnline(nil, "gateway-1") {
		t.Fatal("expected nil online cache to report offline")
	}
}
