package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PauseFlightTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPauseFlightTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseFlightTaskLogic {
	return &PauseFlightTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// PauseFlightTask 暂停当前正在执行的飞行任务。
func (l *PauseFlightTaskLogic) PauseFlightTask(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.PauseFlightTask(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[flight-task] pause flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
