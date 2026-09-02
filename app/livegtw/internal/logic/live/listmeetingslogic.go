package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMeetingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMeetingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingsLogic {
	return &ListMeetingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMeetingsLogic) ListMeetings(req *types.ListMeetingsRequest) (resp *types.ListMeetingsReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.ListMeetings(l.ctx, &live.ListMeetingsReq{
		Status:          req.Status,
		Page:            req.Page,
		PageSize:        req.PageSize,
		CreateTimeStart: req.CreateTimeStart,
		CreateTimeEnd:   req.CreateTimeEnd,
		DeptCode:        req.DeptCode,
		CreateUser:      req.CreateUser,
		Title:           req.Title,
		Identity:        req.Identity,
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
