package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

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
func (l *StopFlightTaskLogic) StopFlightTask(in *djicloud.StopFlightTaskReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.StopFlightTask(l.ctx, in.GetDeviceSn(), &djisdk.FlightTaskStopData{
		FlightID:  in.GetFlightId(),
		WaylineID: int(in.GetWaylineId()),
	})
	if err != nil {
		l.Errorf("[flight-task] stop flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
