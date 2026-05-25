package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
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
	page, pageSize := normalizePage(in.GetPage(), in.GetPageSize())
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiDevice{})
	if in.GatewaySn != "" {
		db = db.Where("gateway_sn = ? OR device_sn IN (?)", in.GatewaySn,
			l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiDeviceTopo{}).
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
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备总数失败")
	}
	var devices []gormmodel.DjiDevice
	if err := db.
		Order("id DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&devices).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备列表失败")
	}
	list := make([]*djicloud.DeviceInfo, 0, len(devices))
	for i := range devices {
		list = append(list, toDeviceInfo(&devices[i]))
	}
	return &djicloud.ListDevicesRes{Total: total, List: list}, nil
}
