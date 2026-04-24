package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExecuteFlightTaskWithOptionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewExecuteFlightTaskWithOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExecuteFlightTaskWithOptionsLogic {
	return &ExecuteFlightTaskWithOptionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ExecuteFlightTaskWithOptionsLogic) ExecuteFlightTaskWithOptions(in *djigateway.FlightTaskWithOptionsReq) (*djigateway.CommonRes, error) {
	prepare := &djisdk.FlightTaskPrepareData{
		FlightID:              in.FlightId,
		TaskType:              int(in.TaskType),
		ExecuteTime:           in.ExecuteTime,
		RthAltitude:           int(in.RthAltitude),
		OutOfControlAction:    int(in.OutOfControlAction),
		ExitWaylineWhenRCLost: int(in.ExitWaylineWhenRcLost),
		File: djisdk.FlightTaskFile{
			URL:         in.WpmlUrl,
			Fingerprint: in.WpmlFingerprint,
		},
	}
	if in.BreakPointIndex >= 0 {
		prepare.BreakPoint = &djisdk.BreakPoint{
			Index:     int(in.BreakPointIndex),
			State:     int(in.BreakPointState),
			Progress:  in.BreakPointProgress,
			WaylineID: int(in.BreakPointWaylineId),
		}
	}

	tid, err := l.svcCtx.DjiClient.ExecuteFlightTaskWithOptions(l.ctx, in.DeviceSn, prepare)
	if err != nil {
		l.Errorf("[flight-task] execute with options failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
