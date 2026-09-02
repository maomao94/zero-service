package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EndMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEndMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndMeetingLogic {
	return &EndMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) error {
	_, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{
		MeetingNo: req.MeetingNo,
	})
	return err
}
