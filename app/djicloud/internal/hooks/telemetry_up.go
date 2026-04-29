package hooks

import (
	"context"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm/clause"
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
		if deviceSn == gatewaySn {
			device.DeviceDomain = gormmodel.DjiDeviceDomainDock
		}
		device.TouchOnline(now)
		updateColumns := []string{"gateway_sn", "is_online", "last_online_at", "update_time"}
		if deviceSn == gatewaySn {
			updateColumns = append(updateColumns, "device_domain")
		}
		if err := gormx.UpsertByColumns(ctx, db, &device, []clause.Column{{Name: "device_sn"}}, updateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert osd device online failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceOsdSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			DataJSON:   toJSONString(data.Data),
			ReportedAt: now,
		}
		if deviceSn == gatewaySn {
			snapshot.DeviceDomain = gormmodel.DjiDeviceDomainDock
		}
		snapshotUpdateColumns := []string{"gateway_sn", "data_json", "reported_at", "update_time"}
		if deviceSn == gatewaySn {
			snapshotUpdateColumns = append(snapshotUpdateColumns, "device_domain")
		}
		if err := gormx.UpsertByColumns(ctx, db, &snapshot, []clause.Column{{Name: "device_sn"}}, snapshotUpdateColumns); err != nil {
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

		device := gormmodel.DjiDevice{
			DeviceSn:  deviceSn,
			GatewaySn: gatewaySn,
		}
		if deviceSn == gatewaySn {
			device.DeviceDomain = gormmodel.DjiDeviceDomainDock
		}
		updateColumns := []string{"gateway_sn", "update_time"}
		if deviceSn == gatewaySn {
			updateColumns = append(updateColumns, "device_domain")
		}
		if err := gormx.UpsertByColumns(ctx, db, &device, []clause.Column{{Name: "device_sn"}}, updateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state device failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceStateSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			DataJSON:   toJSONString(data.Data),
			ReportedAt: now,
		}
		if deviceSn == gatewaySn {
			snapshot.DeviceDomain = gormmodel.DjiDeviceDomainDock
		}
		snapshotUpdateColumns := []string{"gateway_sn", "data_json", "reported_at", "update_time"}
		if deviceSn == gatewaySn {
			snapshotUpdateColumns = append(snapshotUpdateColumns, "device_domain")
		}
		if err := gormx.UpsertByColumns(ctx, db, &snapshot, []clause.Column{{Name: "device_sn"}}, snapshotUpdateColumns); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state snapshot failed: %v", err)
		}
	}
}
