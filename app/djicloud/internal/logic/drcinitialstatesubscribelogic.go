package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcInitialStateSubscribeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcInitialStateSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcInitialStateSubscribeLogic {
	return &DrcInitialStateSubscribeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcInitialStateSubscribeLogic) DrcInitialStateSubscribe(in *djicloud.DrcInitialStateSubscribeReq) (*djicloud.DrcInitialStateSubscribeRes, error) {
	deviceSn := in.GetDeviceSn()
	seq, err := l.svcCtx.DjiClient.DrcNextSeq(deviceSn)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.DjiClient.DrcInitialStateSubscribe(l.ctx, deviceSn, seq); err != nil {
		l.Errorf("[drc] initial state subscribe failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return &djicloud.DrcInitialStateSubscribeRes{Seq: int32(seq)}, nil
}
