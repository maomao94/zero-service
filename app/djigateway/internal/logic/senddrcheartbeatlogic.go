package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendDrcHeartBeatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDrcHeartBeatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDrcHeartBeatLogic {
	return &SendDrcHeartBeatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendDrcHeartBeatLogic) SendDrcHeartBeat(in *djigateway.DrcHeartBeatReq) (*djigateway.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	seq := int(in.GetSeq())
	err := l.svcCtx.DjiClient.SendDrcHeartBeat(l.ctx, deviceSn, seq, in.GetTimestampMillis())
	if err != nil {
		l.Errorf("[drc] send heart beat failed device_sn=%s seq=%d: %v", deviceSn, seq, err)
		return errRes("", err), nil
	}
	return okRes(""), nil
}
