package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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
func (l *DroneFormatLogic) DroneFormat(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DroneFormat(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[remote-debug] drone format failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
