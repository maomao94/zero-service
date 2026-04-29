package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *FlightAuthorityGrabLogic) FlightAuthorityGrab(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.FlightAuthorityGrab(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[drc] flight authority grab failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
