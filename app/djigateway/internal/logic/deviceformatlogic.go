package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

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
func (l *DeviceFormatLogic) DeviceFormat(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DeviceFormat(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] device format failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
