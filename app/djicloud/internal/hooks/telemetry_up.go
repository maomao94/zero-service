package hooks

import (
	"context"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

func NewOsdHandler(db *gormx.DB, onlineCache *collection.Cache) func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	return func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Debugf("[dji-cloud] osd: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)

		gatewaySn := data.Gateway
		if gatewaySn == "" {
			logx.WithContext(ctx).Errorf("[dji-cloud] invalid osd payload: missing gateway, sn=%s tid=%s", deviceSn, data.Tid)
			return
		}
		if onlineCache != nil {
			onlineCache.Set(deviceSn, OnlineValue)
			onlineCache.Set(gatewaySn, OnlineValue)
		}

		now := reportTime(data.Timestamp)
		device := gormmodel.DjiDevice{
			DeviceSn:     deviceSn,
			GatewaySn:    gatewaySn,
			IsOnline:     true,
			LastOnlineAt: sqlNullTime(now),
		}
		device.TouchOnline(now)
		updateColumns := []string{"gateway_sn", "is_online", "last_online_at", "update_time"}
		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] osd storage skipped: db is nil")
			return
		}
		if err := gormx.Upsert(ctx, db, &device, gormx.Columns("device_sn"), updateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert osd device online failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceOsdSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			RawJSON:    toJSONString(data.Data),
			ReportedAt: now,
		}
		snapshotUpdateColumns := []string{"gateway_sn", "raw_json", "reported_at", "update_time"}
		if err := gormx.Upsert(ctx, db, &snapshot, gormx.Columns("device_sn"), snapshotUpdateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert osd snapshot failed: %v", err)
		}
	}
}

func NewStateTelemetryHandler(db *gormx.DB, _ *collection.Cache) func(ctx context.Context, deviceSn string, data *djisdk.StateMessage) {
	return func(ctx context.Context, deviceSn string, data *djisdk.StateMessage) {
		if data == nil {
			return
		}
		logx.WithContext(ctx).Infof("[dji-cloud] state: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)

		gatewaySn := data.Gateway
		if gatewaySn == "" {
			logx.WithContext(ctx).Errorf("[dji-cloud] invalid state payload: missing gateway, sn=%s tid=%s", deviceSn, data.Tid)
			return
		}
		now := reportTime(data.Timestamp)
		versions := extractDeviceVersions(data.Data)

		device := gormmodel.DjiDevice{
			DeviceSn:        deviceSn,
			GatewaySn:       gatewaySn,
			FirmwareVersion: versions.FirmwareVersion,
			HardwareVersion: versions.HardwareVersion,
		}
		updateColumns := appendVersionUpdateColumns([]string{"gateway_sn", "update_time"}, versions)
		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] state storage skipped: db is nil")
			return
		}
		if err := gormx.Upsert(ctx, db, &device, gormx.Columns("device_sn"), updateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state device failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceStateSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			RawJSON:    toJSONString(data.Data),
			ReportedAt: now,
		}
		snapshotUpdateColumns := []string{"gateway_sn", "raw_json", "reported_at", "update_time"}
		if err := gormx.Upsert(ctx, db, &snapshot, gormx.Columns("device_sn"), snapshotUpdateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state snapshot failed: %v", err)
		}
	}
}
