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

func (l *DrcEmergencyLandingLogic) DrcEmergencyLanding(in *djicloud.DrcEmergencyLandingReq) (*djicloud.DrcEmergencyLandingRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.DjiClient.DrcEmergencyLanding(l.ctx, deviceSn, seq); err != nil {
		return nil, err
	}
	return &djicloud.DrcEmergencyLandingRes{Seq: int32(seq)}, nil
}
