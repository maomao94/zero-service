package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

type KickParticipantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickParticipantLogic {
	return &KickParticipantLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 踢人（移除指定参与者）
func (l *KickParticipantLogic) KickParticipant(in *live.KickParticipantReq) (*live.KickParticipantRes, error) {
	if err := requireMeetingIdentity(in.MeetingNo, in.Identity); err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo); err != nil {
		return nil, meetingErr(err)
	}
	if _, err := l.svcCtx.LiveKit.Room().RemoveParticipant(l.ctx, &livekit.RoomParticipantIdentity{Room: in.MeetingNo, Identity: in.Identity}); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "移除参与者失败")
	}
	l.Logger.Infof("participant kicked: meeting=%s identity=%s", in.MeetingNo, in.Identity)
	return &live.KickParticipantRes{}, nil
}
