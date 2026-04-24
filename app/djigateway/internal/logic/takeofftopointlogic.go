package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type TakeoffToPointLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTakeoffToPointLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TakeoffToPointLogic {
	return &TakeoffToPointLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TakeoffToPointLogic) TakeoffToPoint(in *djigateway.TakeoffToPointReq) (*djigateway.CommonRes, error) {
	data := &djisdk.TakeoffToPointData{
		FlightID:              in.FlightId,
		TargetLatitude:        in.TargetLatitude,
		TargetLongitude:       in.TargetLongitude,
		TargetHeight:          in.TargetHeight,
		SecurityTakeoffHeight: in.SecurityTakeoffHeight,
		RthAltitude:           in.RthAltitude,
		RCLostAction:          int(in.RcLostAction),
		MaxSpeed:              in.MaxSpeed,
		CommanderFlightHeight: in.CommanderFlightHeight,
	}
	tid, err := l.svcCtx.DjiClient.TakeoffToPoint(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] takeoff to point failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
