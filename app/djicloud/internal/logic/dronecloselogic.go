package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DroneCloseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDroneCloseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DroneCloseLogic {
	return &DroneCloseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DroneClose 关闭机巢中的无人机电源。
func (l *DroneCloseLogic) DroneClose(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DroneClose(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] drone close failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
