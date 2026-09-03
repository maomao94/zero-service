package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateMeetingTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGenerateMeetingTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateMeetingTicketLogic {
	return &GenerateMeetingTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GenerateMeetingTicketLogic) GenerateMeetingTicket(req *types.GenerateMeetingTicketRequest) (resp *types.GenerateMeetingTicketReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
		MeetingNo:         req.MeetingNo,
		MeetingCode:       req.MeetingCode,
		Identity:          req.Identity,
		Name:              req.Name,
		ExpireSeconds:     uint32(req.ExpireSeconds),
		CanPublish:        req.CanPublish,
		CanSubscribe:      req.CanSubscribe,
		CanPublishData:    req.CanPublishData,
		CanPublishSources: req.CanPublishSources,
		TicketType:        req.TicketType,
	})
	if err != nil {
		return nil, err
	}
	return &types.GenerateMeetingTicketReply{
		Ticket:            r.GetTicket(),
		ExpireTime:        r.GetExpireTime(),
		CanPublish:        r.GetCanPublish(),
		CanSubscribe:      r.GetCanSubscribe(),
		CanPublishData:    r.GetCanPublishData(),
		CanPublishSources: r.GetCanPublishSources(),
	}, nil
}
