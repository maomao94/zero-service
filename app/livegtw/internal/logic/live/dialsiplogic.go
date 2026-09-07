package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DialSipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDialSipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DialSipLogic {
	return &DialSipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DialSipLogic) DialSip(req *types.DialSipRequest) (resp *types.DialSipReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.DialSip(l.ctx, &live.DialSipReq{
		CalleeNumber:    req.CalleeNumber,
		MeetingNo:       req.MeetingNo,
		ParticipantName: req.ParticipantName,
		ProviderCode:    req.ProviderCode,
	})
	if err != nil {
		return nil, err
	}
	return &types.DialSipReply{
		Meeting:   toMeetingInfo(r.GetMeeting()),
		SipCallId: r.GetSipCallId(),
	}, nil
}
