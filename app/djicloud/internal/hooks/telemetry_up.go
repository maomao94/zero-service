package hooks

import (
	"context"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

func NewOsdHandler(db *gormx.DB, onlineCache *collection.Cache, pushCli socketpush.SocketPushClient, disableSQLTrace bool) func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	return func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
		if data == nil {
			return
		}
		if disableSQLTrace {
			ctx = gormx.WithoutSQLTrace(ctx)
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
		updateData := map[string]any{
			"gateway_sn":     device.GatewaySn,
			"is_online":      device.IsOnline,
			"last_online_at": device.LastOnlineAt,
		}
		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] osd storage skipped: db is nil")
			return
		}
		if err := gormx.UpdateOrCreate(ctx, db, &gormmodel.DjiDevice{}, map[string]any{"device_sn": deviceSn}, &device, updateData); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert osd device online failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceOsdSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			RawJSON:    toJSONString(data.Data),
			ReportedAt: now,
		}
		snapshotUpdateData := map[string]any{
			"gateway_sn":  snapshot.GatewaySn,
			"raw_json":    snapshot.RawJSON,
			"reported_at": snapshot.ReportedAt,
		}
		if err := gormx.UpdateOrCreate(ctx, db, &gormmodel.DjiDeviceOsdSnapshot{}, map[string]any{"device_sn": deviceSn}, &snapshot, snapshotUpdateData); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert osd snapshot failed: %v", err)
		}

		if pushCli != nil {
			pushCtx := context.WithoutCancel(ctx)
			threading.GoSafe(func() {
				reqId, _ := tool.SimpleUUID()
				room := "thing/product/" + deviceSn + "/osd"
				_, err := pushCli.BroadcastRoom(pushCtx, &socketpush.BroadcastRoomReq{
					ReqId:   reqId,
					Room:    room,
					Event:   "telemetry:osd",
					Payload: toJSONString(data.Data),
				})
				if err != nil {
					logx.WithContext(pushCtx).Errorf("[dji-cloud] socket push osd failed: sn=%s err=%v", deviceSn, err)
				}
			})
		}
	}
}

func NewStateTelemetryHandler(db *gormx.DB, _ *collection.Cache, pushCli socketpush.SocketPushClient) func(ctx context.Context, deviceSn string, data *djisdk.StateMessage) {
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
		updateData := map[string]any{"gateway_sn": device.GatewaySn}
		if versions.FirmwareVersion != "" {
			updateData["firmware_version"] = versions.FirmwareVersion
		}
		if versions.HardwareVersion != "" {
			updateData["hardware_version"] = versions.HardwareVersion
		}
		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] state storage skipped: db is nil")
			return
		}
		if err := gormx.UpdateOrCreate(ctx, db, &gormmodel.DjiDevice{}, map[string]any{"device_sn": deviceSn}, &device, updateData); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state device failed: %v", err)
		}

		snapshot := gormmodel.DjiDeviceStateSnapshot{
			DeviceSn:   deviceSn,
			GatewaySn:  gatewaySn,
			RawJSON:    toJSONString(data.Data),
			ReportedAt: now,
		}
		snapshotUpdateData := map[string]any{
			"gateway_sn":  snapshot.GatewaySn,
			"raw_json":    snapshot.RawJSON,
			"reported_at": snapshot.ReportedAt,
		}
		if err := gormx.UpdateOrCreate(ctx, db, &gormmodel.DjiDeviceStateSnapshot{}, map[string]any{"device_sn": deviceSn}, &snapshot, snapshotUpdateData); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] upsert state snapshot failed: %v", err)
		}

		if pushCli != nil {
			pushCtx := context.WithoutCancel(ctx)
			threading.GoSafe(func() {
				reqId, _ := tool.SimpleUUID()
				room := "thing/product/" + deviceSn + "/state"
				_, err := pushCli.BroadcastRoom(pushCtx, &socketpush.BroadcastRoomReq{
					ReqId:   reqId,
					Room:    room,
					Event:   "telemetry:state",
					Payload: toJSONString(data.Data),
				})
				if err != nil {
					logx.WithContext(pushCtx).Errorf("[dji-cloud] socket push state failed: sn=%s err=%v", deviceSn, err)
				}
			})
		}
	}
}
