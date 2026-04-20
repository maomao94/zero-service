package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExecuteFlightTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewExecuteFlightTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExecuteFlightTaskLogic {
	return &ExecuteFlightTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ExecuteFlightTaskLogic) ExecuteFlightTask(in *djigateway.FlightTaskReq) (*djigateway.CommonRes, error) {
	l.Infof("[flight-task] device_sn=%s task_id=%s wpml_url=%s", in.DeviceSn, in.TaskId, in.WpmlUrl)

	tid, err := l.svcCtx.DjiClient.ExecuteFlightTask(l.ctx, in.DeviceSn, in.TaskId, in.WpmlUrl)
	if err != nil {
		l.Errorf("[flight-task] execute failed: %v", err)
		return &djigateway.CommonRes{
			Code:    -1,
			Message: err.Error(),
			Tid:     tid,
		}, nil
	}

	return &djigateway.CommonRes{
		Code:    0,
		Message: "success",
		Tid:     tid,
	}, nil
}
