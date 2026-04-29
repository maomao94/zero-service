package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceFormatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeviceFormatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceFormatLogic {
	return &DeviceFormatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeviceFormat 格式化机巢设备存储。
func (l *DeviceFormatLogic) DeviceFormat(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DeviceFormat(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] device format failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
