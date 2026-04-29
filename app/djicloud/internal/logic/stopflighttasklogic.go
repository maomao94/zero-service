package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopFlightTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopFlightTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopFlightTaskLogic {
	return &StopFlightTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// StopFlightTask 强制停止当前航线任务。
func (l *StopFlightTaskLogic) StopFlightTask(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.StopFlightTask(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[flight-task] stop flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
