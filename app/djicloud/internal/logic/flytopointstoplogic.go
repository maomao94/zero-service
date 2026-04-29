package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlyToPointStopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlyToPointStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlyToPointStopLogic {
	return &FlyToPointStopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FlyToPointStopLogic) FlyToPointStop(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlyToPointStop(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] fly to point stop failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
