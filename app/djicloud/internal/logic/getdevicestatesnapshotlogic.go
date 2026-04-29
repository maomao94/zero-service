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
func (l *GetDeviceStateSnapshotLogic) GetDeviceStateSnapshot(in *djicloud.DeviceSnReq) (*djicloud.DeviceStateSnapshotRes, error) {
	var item gormmodel.DjiDeviceStateSnapshot
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("device_sn = ?", in.DeviceSn).First(&item).Error; err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &djicloud.DeviceStateSnapshotRes{Data: toStateSnapshot(&item)}, nil
}
