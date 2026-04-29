package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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
func (l *PauseFlightTaskLogic) PauseFlightTask(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.PauseFlightTask(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[flight-task] pause flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
