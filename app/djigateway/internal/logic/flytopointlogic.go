package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlyToPointLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlyToPointLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlyToPointLogic {
	return &FlyToPointLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FlyToPointLogic) FlyToPoint(in *djigateway.FlyToPointReq) (*djigateway.CommonRes, error) {
	points := make([]djisdk.FlyToWaypoint, 0, len(in.Points))
	for _, p := range in.Points {
		points = append(points, djisdk.FlyToWaypoint{
			Latitude:  p.Latitude,
			Longitude: p.Longitude,
			Height:    p.Height,
		})
	}
	data := &djisdk.FlyToPointData{
		MaxSpeed: in.MaxSpeed,
		FlyToID:  in.FlyToId,
		Points:   points,
	}
	tid, err := l.svcCtx.DjiClient.FlyToPoint(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] fly to point failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
