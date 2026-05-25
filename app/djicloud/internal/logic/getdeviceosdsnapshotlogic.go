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

type GetDeviceOsdSnapshotLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeviceOsdSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceOsdSnapshotLogic {
	return &GetDeviceOsdSnapshotLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetDeviceOsdSnapshot 查询设备最近一次 OSD 遥测快照。
func (l *GetDeviceOsdSnapshotLogic) GetDeviceOsdSnapshot(in *djicloud.GetDeviceOsdSnapshotReq) (*djicloud.DeviceOsdSnapshotRes, error) {
	var item gormmodel.DjiDeviceOsdSnapshot
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("device_sn = ?", in.DeviceSn).First(&item).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_RECORD_NOT_EXIST, err, "查询设备OSD快照失败")
	}
	return &djicloud.DeviceOsdSnapshotRes{Data: toOsdSnapshot(&item)}, nil
}
