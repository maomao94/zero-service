package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendMeetingDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendMeetingDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMeetingDataLogic {
	return &SendMeetingDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendMeetingDataLogic) SendMeetingData(req *types.SendMeetingDataRequest) error {
	_, err := l.svcCtx.LiveRpcCli.SendMeetingData(l.ctx, &live.SendMeetingDataReq{
		MeetingNo:    req.MeetingNo,
		Topic:        req.Topic,
		Payload:      []byte(req.Payload),
		Destinations: req.Destinations,
	})
	return err
}
