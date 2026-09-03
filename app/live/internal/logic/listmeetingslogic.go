package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"

	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMeetingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMeetingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingsLogic {
	return &ListMeetingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询会议列表
func (l *ListMeetingsLogic) ListMeetings(in *live.ListMeetingsReq) (*live.ListMeetingsRes, error) {
	page := in.Page
	pageSize := in.PageSize

	meetings, total, err := l.svcCtx.MeetingRepo.ListMeetings(l.ctx, &svc.MeetingListQuery{
		Status:          in.Status,
		Page:            page,
		PageSize:        pageSize,
		CreateTimeStart: in.CreateTimeStart,
		CreateTimeEnd:   in.CreateTimeEnd,
		DeptCode:        in.DeptCode,
		CreateUser:      in.CreateUser,
		Title:           in.Title,
		Identity:        in.Identity,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询会议列表失败")
	}
	resp := &live.ListMeetingsRes{Total: total}
	for i := range meetings {
		resp.Meetings = append(resp.Meetings, toMeetingInfo(&meetings[i]))
	}
	return resp, nil
}
