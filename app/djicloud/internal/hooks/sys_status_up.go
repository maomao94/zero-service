package hooks

import (
	"context"
	"encoding/json"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm/clause"
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

		if db == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status update topo skipped: db is nil")
			return djisdk.PlatformResultOK
		}

		if err := db.WithContext(ctx).Transact(func(tx *gormx.DB) error {
			if err := gormx.Upsert(ctx, tx, &gormmodel.DjiDevice{
				DeviceSn:  gatewaySn,
				GatewaySn: gatewaySn,
			}, []clause.Column{{Name: "device_sn"}}, []string{"gateway_sn", "update_time"}); err != nil {
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
				subDomain := sub.Domain
				if err := gormx.Upsert(ctx, tx, &gormmodel.DjiDeviceTopo{
					GatewaySn:        gatewaySn,
					SubDeviceSn:      sub.SN,
					Domain:           subDomain,
					SubDeviceType:    sub.Type,
					SubDeviceSubType: sub.SubType,
					SubDeviceIndex:   sub.Index,
					ThingVersion:     sub.ThingVersion,
				}, []clause.Column{{Name: "gateway_sn"}, {Name: "sub_device_sn"}}, []string{"domain", "sub_device_type", "sub_device_sub_type", "sub_device_index", "thing_version", "update_time"}); err != nil {
					return err
				}

				if err := gormx.Upsert(ctx, tx, &gormmodel.DjiDevice{
					DeviceSn:  sub.SN,
					GatewaySn: gatewaySn,
				}, []clause.Column{{Name: "device_sn"}}, []string{"gateway_sn", "update_time"}); err != nil {
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
