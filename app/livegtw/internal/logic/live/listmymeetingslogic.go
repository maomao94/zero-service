package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/authctx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyMeetingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyMeetingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyMeetingsLogic {
	return &ListMyMeetingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyMeetingsLogic) ListMyMeetings(req *types.ListMyMeetingsRequest) (resp *types.ListMeetingsReply, err error) {
	identity := authctx.GetUserId(l.ctx)
	r, err := l.svcCtx.LiveRpcCli.ListMeetings(l.ctx, &live.ListMeetingsReq{
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
		Identity: identity,
	})
	if err != nil {
		return nil, err
	}
	meetings := make([]types.MeetingInfo, 0, len(r.GetMeetings()))
	for _, m := range r.GetMeetings() {
		meetings = append(meetings, toMeetingInfo(m))
	}
	return &types.ListMeetingsReply{
		Meetings: meetings,
		Total:    r.GetTotal(),
	}, nil
}
