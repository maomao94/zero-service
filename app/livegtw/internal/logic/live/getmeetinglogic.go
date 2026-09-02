package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingLogic {
	return &GetMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeetingLogic) GetMeeting(req *types.GetMeetingRequest) (resp *types.GetMeetingReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.GetMeeting(l.ctx, &live.GetMeetingReq{
		MeetingNo: req.MeetingNo,
	})
	if err != nil {
		return nil, err
	}
	return &types.GetMeetingReply{
		Meeting: toMeetingInfo(r.GetMeeting()),
	}, nil
}
