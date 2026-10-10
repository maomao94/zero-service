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

type ListMeetingRecordingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询会议录制列表
func NewListMeetingRecordingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingRecordingsLogic {
	return &ListMeetingRecordingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMeetingRecordingsLogic) ListMeetingRecordings(req *types.ListMeetingRecordingsRequest) (resp *types.ListMeetingRecordingsReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.ListMeetingRecordings(l.ctx, &live.ListMeetingRecordingsReq{
		MeetingNo: req.MeetingNo,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	recordings := make([]types.MeetingRecordingInfo, 0, len(r.GetRecordings()))
	for _, rec := range r.GetRecordings() {
		recordings = append(recordings, toRecordingInfo(rec))
	}
	return &types.ListMeetingRecordingsReply{
		Recordings: recordings,
		Total:      r.GetTotal(),
	}, nil
}
