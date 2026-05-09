package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcForceLandingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcForceLandingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcForceLandingLogic {
	return &DrcForceLandingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcForceLandingLogic) DrcForceLanding(in *djicloud.DrcForceLandingReq) (*djicloud.DrcForceLandingRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DrcManager.GetNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.DjiClient.DrcForceLanding(l.ctx, deviceSn, seq); err != nil {
		l.Errorf("[drc] force landing failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcForceLandingRes{Seq: int32(seq)}, nil
}
