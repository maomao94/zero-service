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

type GetDeviceStateSnapshotLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeviceStateSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceStateSnapshotLogic {
	return &GetDeviceStateSnapshotLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetDeviceStateSnapshot 查询设备最近一次 State 状态快照。
func (l *GetDeviceStateSnapshotLogic) GetDeviceStateSnapshot(in *djicloud.GetDeviceStateSnapshotReq) (*djicloud.DeviceStateSnapshotRes, error) {
	var item gormmodel.DjiDeviceStateSnapshot
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("device_sn = ?", in.DeviceSn).First(&item).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_RECORD_NOT_EXIST, err, "查询设备State快照失败")
	}
	return &djicloud.DeviceStateSnapshotRes{Data: toStateSnapshot(&item)}, nil
}
