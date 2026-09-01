package logic

import (
	"context"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeetingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingLogic {
	return &GetMeetingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询会议详情
func (l *GetMeetingLogic) GetMeeting(in *live.GetMeetingReq) (*live.GetMeetingRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, strings.TrimSpace(in.MeetingNo))
	if err != nil {
		return nil, meetingErr(err)
	}
	return &live.GetMeetingRes{Meeting: toMeetingInfo(meeting)}, nil
}
