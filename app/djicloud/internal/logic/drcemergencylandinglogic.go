package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcEmergencyLandingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcEmergencyLandingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcEmergencyLandingLogic {
	return &DrcEmergencyLandingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcEmergencyLandingLogic) DrcEmergencyLanding(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.DrcEmergencyLanding(l.ctx, in.GetDeviceSn())
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
