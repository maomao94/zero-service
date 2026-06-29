package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcNightLightsStateSetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcNightLightsStateSetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcNightLightsStateSetLogic {
	return &DrcNightLightsStateSetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcNightLightsStateSetLogic) DrcNightLightsStateSet(in *djicloud.DrcNightLightsStateSetReq) (*djicloud.DrcNightLightsStateSetRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	data := &djisdk.DrcNightLightsStateSetData{NightLightsState: int(in.GetNightLightsState())}
	if _, err := l.svcCtx.DjiClient.DrcNightLightsStateSet(l.ctx, deviceSn, seq, data); err != nil {
		return nil, err
	}
	return &djicloud.DrcNightLightsStateSetRes{Seq: int32(seq)}, nil
}
