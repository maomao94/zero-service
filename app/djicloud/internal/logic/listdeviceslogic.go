package logic

import (
	"context"
	"errors"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"gorm.io/gorm"
)

type ListDevicesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDevicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDevicesLogic {
	return &ListDevicesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListDevices 查询设备列表，包含机巢、无人机、负载等设备。
func (l *ListDevicesLogic) ListDevices(in *djicloud.ListDevicesReq) (*djicloud.ListDevicesRes, error) {
	baseDB := l.svcCtx.DB.WithContext(l.ctx)
	db := baseDB.Model(&gormmodel.DjiDevice{})
	if in.GatewaySn != "" {
		db = db.Where("gateway_sn = ? OR device_sn IN (?)", in.GatewaySn,
			l.svcCtx.DB.Model(&gormmodel.DjiDeviceTopo{}).
				Select("sub_device_sn").
				Where("gateway_sn = ?", in.GatewaySn),
		)
	}
	if in.OnlineStatus == 1 || in.OnlineStatus == 2 {
		db = db.Where("is_online = ?", in.OnlineStatus == 1)
	}
	if in.Keyword != "" {
		db = db.Where("device_sn LIKE ? OR alias LIKE ?", "%"+in.Keyword+"%", "%"+in.Keyword+"%")
	}
	var devices []gormmodel.DjiDevice
	pageResult, err := gormx.QueryPage(db.Order("id DESC"), int(in.GetPage()), int(in.GetPageSize()), &devices)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备列表失败")
	}

	list := make([]*djicloud.DeviceListItem, 0, len(devices))
	for i := range devices {
		sn := devices[i].DeviceSn
		gw := devices[i].GatewaySn
		item := &djicloud.DeviceListItem{Device: toDeviceInfo(&devices[i])}

		var osd *gormmodel.DjiDeviceOsdSnapshot
		var state *gormmodel.DjiDeviceStateSnapshot
		var topos []gormmodel.DjiDeviceTopo
		var task *gormmodel.DjiDockDeviceFlightTaskState

		if err := mr.Finish(
			func() error {
				var v gormmodel.DjiDeviceOsdSnapshot
				if err := baseDB.
					Where("device_sn = ?", sn).First(&v).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				osd = &v
				return nil
			},
			func() error {
				var v gormmodel.DjiDeviceStateSnapshot
				if err := baseDB.
					Where("device_sn = ?", sn).First(&v).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				state = &v
				return nil
			},
			func() error {
				return baseDB.
					Where("gateway_sn = ? OR sub_device_sn = ?", sn, sn).
					Order("update_time DESC, id DESC").
					Find(&topos).Error
			},
			func() error {
				if gw == "" {
					return nil
				}
				var v gormmodel.DjiDockDeviceFlightTaskState
				if err := baseDB.
					Where("gateway_sn = ?", gw).First(&v).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				task = &v
				return nil
			},
		); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备列表失败")
		}

		if osd != nil {
			item.Osd = toTelemetrySnapshotBrief(osd)
		}
		if state != nil {
			item.State = toTelemetrySnapshotBriefFromState(state)
		}
		if len(topos) > 0 {
			item.Topo = toTopoInfoList(topos)
		}
		if task != nil {
			item.FlightTaskState = toDockFlightTaskStateInfo(task)
		}
		list = append(list, item)
	}
	return &djicloud.ListDevicesRes{Total: pageResult.Total, List: list}, nil
}
