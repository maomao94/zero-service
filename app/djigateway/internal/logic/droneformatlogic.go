package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DroneFormatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDroneFormatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DroneFormatLogic {
	return &DroneFormatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DroneFormat 格式化无人机存储。
func (l *DroneFormatLogic) DroneFormat(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DroneFormat(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] drone format failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
