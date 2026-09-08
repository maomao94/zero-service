// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyMeetingParticipantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 通知用户加入会议
func NewNotifyMeetingParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyMeetingParticipantLogic {
	return &NotifyMeetingParticipantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NotifyMeetingParticipantLogic) NotifyMeetingParticipant(req *types.NotifyMeetingParticipantRequest) (resp *types.NotifyMeetingParticipantReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.NotifyMeetingParticipant(l.ctx, &live.NotifyMeetingParticipantReq{
		MeetingNo:   req.MeetingNo,
		MeetingCode: req.MeetingCode,
		Identity:    req.Identity,
	})
	if err != nil {
		return nil, err
	}
	return &types.NotifyMeetingParticipantReply{RequestId: r.GetRequestId()}, nil
}
