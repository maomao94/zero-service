package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceRebootLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeviceRebootLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRebootLogic {
	return &DeviceRebootLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeviceReboot 重启机巢设备。
func (l *DeviceRebootLogic) DeviceReboot(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DeviceReboot(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] device reboot failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
