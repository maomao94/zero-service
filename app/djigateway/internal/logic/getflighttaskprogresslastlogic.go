package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/hooks"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFlightTaskProgressLastLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFlightTaskProgressLastLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFlightTaskProgressLastLogic {
	return &GetFlightTaskProgressLastLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetFlightTaskProgressLast 查询本服务内存中最近一次 flighttask_progress 上报。
func (l *GetFlightTaskProgressLastLogic) GetFlightTaskProgressLast(in *djigateway.DeviceSnReq) (*djigateway.FlightTaskProgressLastRes, error) {
	has, at, js := hooks.GetFlightTaskProgressLast(l.svcCtx.FlightProgressCache, in.DeviceSn)
	return &djigateway.FlightTaskProgressLastRes{
		HasProgress:  has,
		CachedAtMs:   at,
		ProgressJson: js,
	}, nil
}
