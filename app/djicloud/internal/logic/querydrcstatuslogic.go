package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryDrcStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryDrcStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryDrcStatusLogic {
	return &QueryDrcStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryDrcStatusLogic) QueryDrcStatus(in *djicloud.QueryDrcStatusReq) (*djicloud.DrcStatusRes, error) {
	deviceSn := in.GetDeviceSn()
	enabled, startedAt, lastHb, nextSeq, alive := l.svcCtx.DrcManager.GetStatus(deviceSn)
	return &djicloud.DrcStatusRes{
		Enabled:                   enabled,
		StartedAtMillis:           timeMillis(startedAt),
		LastDeviceHeartbeatMillis: timeMillis(lastHb),
		NextSeq:                   int32(nextSeq),
		IsAlive:                   alive,
	}, nil
}
