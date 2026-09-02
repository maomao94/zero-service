package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMeetingLogic {
	return &CreateMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingRequest) (resp *types.CreateMeetingReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.CreateMeeting(l.ctx, &live.CreateMeetingReq{
		Title: req.Title,
	})
	if err != nil {
		return nil, err
	}
	return &types.CreateMeetingReply{
		Meeting: toMeetingInfo(r.GetMeeting()),
	}, nil
}
