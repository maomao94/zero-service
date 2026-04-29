package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/hooks"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type IsDeviceOnlineLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIsDeviceOnlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsDeviceOnlineLogic {
	return &IsDeviceOnlineLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// IsDeviceOnline 查询机巢在线状态。
func (l *IsDeviceOnlineLogic) IsDeviceOnline(in *djicloud.DeviceSnReq) (*djicloud.DeviceOnlineRes, error) {
	online := hooks.IsOnline(l.svcCtx.OnlineCache, in.DeviceSn)
	return &djicloud.DeviceOnlineRes{IsOnline: online}, nil
}
