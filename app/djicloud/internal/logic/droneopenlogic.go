package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DroneOpenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDroneOpenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DroneOpenLogic {
	return &DroneOpenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DroneOpen 开启机巢中的无人机电源。
func (l *DroneOpenLogic) DroneOpen(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DroneOpen(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] drone open failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
