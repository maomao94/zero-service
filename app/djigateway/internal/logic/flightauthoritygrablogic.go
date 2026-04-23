package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlightAuthorityGrabLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlightAuthorityGrabLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlightAuthorityGrabLogic {
	return &FlightAuthorityGrabLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FlightAuthorityGrabLogic) FlightAuthorityGrab(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlightAuthorityGrab(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] flight authority grab failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
