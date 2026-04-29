package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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

func (l *SendDrcHeartBeatLogic) SendDrcHeartBeat(in *djicloud.DrcHeartBeatReq) (*djicloud.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
	seq := int(in.GetSeq())
	tid, err := l.svcCtx.DjiClient.SendDrcHeartBeat(l.ctx, deviceSn, seq, in.GetTimestampMillis())
	if err != nil {
		l.Errorf("[drc] send heart beat failed device_sn=%s seq=%d tid=%s: %v", deviceSn, seq, tid, err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
