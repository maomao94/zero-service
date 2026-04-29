package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelFlightTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelFlightTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelFlightTaskLogic {
	return &CancelFlightTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CancelFlightTask 取消指定的飞行任务。
func (l *CancelFlightTaskLogic) CancelFlightTask(in *djicloud.CancelFlightTaskReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.CancelFlightTask(l.ctx, in.DeviceSn, in.FlightIds)
	if err != nil {
		l.Errorf("[flight-task] cancel flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
