package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/authctx"

	"github.com/zeromicro/go-zero/core/logx"
)

type JoinMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewJoinMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMeetingLogic {
	return &JoinMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JoinMeetingLogic) JoinMeeting(req *types.JoinMeetingRequest) (resp *types.JoinMeetingReply, err error) {
	identity := authctx.GetUserId(l.ctx)
	name := authctx.GetUserName(l.ctx)
	r, err := l.svcCtx.LiveRpcCli.JoinMeeting(l.ctx, &live.JoinMeetingReq{
		MeetingNo:         req.MeetingNo,
		MeetingCode:       req.MeetingCode,
		Identity:          identity,
		Name:              name,
		CanPublish:        req.CanPublish,
		CanSubscribe:      req.CanSubscribe,
		CanPublishData:    req.CanPublishData,
		CanPublishSources: req.CanPublishSources,
	})
	if err != nil {
		return nil, err
	}
	return &types.JoinMeetingReply{
		Token:             r.GetToken(),
		WsUrl:             r.GetWsUrl(),
		Meeting:           toMeetingInfo(r.GetMeeting()),
		CanPublish:        r.GetCanPublish(),
		CanSubscribe:      r.GetCanSubscribe(),
		CanPublishData:    r.GetCanPublishData(),
		CanPublishSources: r.GetCanPublishSources(),
	}, nil
}
