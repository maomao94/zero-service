package hooks

import (
	"context"
	"encoding/json"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

func NewStatusHandler(db *gormx.DB, _ *collection.Cache) djisdk.StatusHandler {
	return func(ctx context.Context, gatewaySn string, data *djisdk.StatusMessage) int {
		if data == nil {
			return djisdk.PlatformResultOK
		}
		logx.WithContext(ctx).Infof("[dji-cloud] status: sn=%s method=%s", gatewaySn, data.Method)

		if data.Method != djisdk.MethodUpdateTopo {
			return djisdk.PlatformResultOK
		}

		raw, err := json.Marshal(data.Data)
		if err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status marshal data failed: %v", err)
			return djisdk.PlatformResultHandlerError
		}
		var topo djisdk.TopoUpdateData
		if err := json.Unmarshal(raw, &topo); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status unmarshal topo failed: %v", err)
			return djisdk.PlatformResultHandlerError
		}
		now := reportTime(data.Timestamp)
		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status update topo skipped: db is nil")
			return djisdk.PlatformResultOK
		}

		if err := db.WithContext(ctx).Transact(func(tx *gormx.DB) error {
			gatewayDevice := gormmodel.DjiDevice{
				DeviceSn:     gatewaySn,
				GatewaySn:    gatewaySn,
				IsOnline:     true,
				LastOnlineAt: sqlNullTime(now),
			}
			if err := gormx.UpdateOrCreate(tx.WithContext(ctx), &gormmodel.DjiDevice{}, map[string]any{"device_sn": gatewaySn}, &gatewayDevice, map[string]any{"gateway_sn": gatewaySn}); err != nil {
				return err
			}

			reportedSubDevices := make([]string, 0, len(topo.SubDevices))
			for _, sub := range topo.SubDevices {
				if sub.SN != "" {
					reportedSubDevices = append(reportedSubDevices, sub.SN)
				}
			}
			clearDB := tx.WithContext(ctx).Where("gateway_sn = ?", gatewaySn)
			if len(reportedSubDevices) > 0 {
				clearDB = clearDB.Where("sub_device_sn NOT IN ?", reportedSubDevices)
			}
			if err := clearDB.Delete(&gormmodel.DjiDeviceTopo{}).Error; err != nil {
				return err
			}

			for _, sub := range topo.SubDevices {
				if err := gormx.Restore(tx.DB, &gormmodel.DjiDeviceTopo{}, "gateway_sn = ? AND sub_device_sn = ?", gatewaySn, sub.SN); err != nil {
					return err
				}
				subDomain := sub.Domain
				topoRecord := gormmodel.DjiDeviceTopo{
					GatewaySn:        gatewaySn,
					SubDeviceSn:      sub.SN,
					Domain:           subDomain,
					SubDeviceType:    sub.Type,
					SubDeviceSubType: sub.SubType,
					SubDeviceIndex:   sub.Index,
					ThingVersion:     sub.ThingVersion,
				}
				topoUpdateData := map[string]any{
					"domain":              topoRecord.Domain,
					"sub_device_type":     topoRecord.SubDeviceType,
					"sub_device_sub_type": topoRecord.SubDeviceSubType,
					"sub_device_index":    topoRecord.SubDeviceIndex,
					"thing_version":       topoRecord.ThingVersion,
				}
				if err := gormx.UpdateOrCreate(tx.WithContext(ctx), &gormmodel.DjiDeviceTopo{}, map[string]any{"gateway_sn": gatewaySn, "sub_device_sn": sub.SN}, &topoRecord, topoUpdateData); err != nil {
					return err
				}

				subDevice := gormmodel.DjiDevice{
					DeviceSn:  sub.SN,
					GatewaySn: gatewaySn,
				}
				if err := gormx.UpdateOrCreate(tx.WithContext(ctx), &gormmodel.DjiDevice{}, map[string]any{"device_sn": sub.SN}, &subDevice, map[string]any{"gateway_sn": gatewaySn}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status update topo failed: %v", err)
			return djisdk.PlatformResultHandlerError
		}

		logx.WithContext(ctx).Infof("[dji-cloud] topo update: sn=%s sub_devices=%d", gatewaySn, len(topo.SubDevices))
		return djisdk.PlatformResultOK
	}
}
