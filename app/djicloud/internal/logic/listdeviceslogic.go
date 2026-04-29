package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if in.DeviceDomainFilter != "" {
		db = db.Where("device_domain = ?", in.DeviceDomainFilter)
	}
	if in.OnlineStatus == 1 || in.OnlineStatus == 2 {
		db = db.Where("is_online = ?", in.OnlineStatus == 1)
	}
	if in.Keyword != "" {
		db = db.Where("device_sn LIKE ? OR alias LIKE ?", "%"+in.Keyword+"%", "%"+in.Keyword+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var devices []gormmodel.DjiDevice
	if err := db.Order("id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&devices).Error; err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	list := make([]*djicloud.DeviceInfo, 0, len(devices))
	for i := range devices {
		list = append(list, toDeviceInfo(&devices[i]))
	}
	return &djicloud.ListDevicesRes{Total: total, List: list}, nil
}
