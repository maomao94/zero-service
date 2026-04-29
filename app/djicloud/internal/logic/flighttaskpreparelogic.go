package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlightTaskPrepareLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlightTaskPrepareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlightTaskPrepareLogic {
	return &FlightTaskPrepareLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FlightTaskPrepareLogic) FlightTaskPrepare(in *djicloud.FlightTaskPrepareReq) (*djicloud.CommonRes, error) {
	file := in.GetFile()
	if file == nil {
		file = &djicloud.FlightTaskFileRef{}
	}
	prepare := &djisdk.FlightTaskPrepareData{
		FlightID:              in.FlightId,
		TaskType:              int(in.TaskType),
		ExecuteTime:           in.ExecuteTime,
		WaylineType:           int(in.WaylineType),
		File:                  djisdk.FlightTaskFile{URL: file.GetUrl(), Fingerprint: file.GetFingerprint()},
		RthAltitude:           int(in.RthAltitude),
		OutOfControlAction:    int(in.OutOfControlAction),
		ExitWaylineWhenRCLost: int(in.ExitWaylineWhenRcLost),
	}
	if bp := in.GetBreakPoint(); bp != nil {
		prepare.BreakPoint = &djisdk.BreakPoint{
			Index:     int(bp.Index),
			State:     int(bp.State),
			Progress:  bp.Progress,
			WaylineID: int(bp.WaylineId),
		}
	}
	if sm := in.GetSimulateMission(); sm != nil && sm.IsEnable {
		prepare.SimulateMission = &djisdk.SimulateMission{
			IsEnable:  sm.IsEnable,
			Latitude:  sm.Latitude,
			Longitude: sm.Longitude,
		}
	}

	tid, err := l.svcCtx.DjiClient.FlightTaskPrepare(l.ctx, in.DeviceSn, prepare)
	if err != nil {
		l.Errorf("[flight-task] flighttask_prepare failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
