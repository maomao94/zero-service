package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMeetingRecordingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMeetingRecordingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingRecordingsLogic {
	return &ListMeetingRecordingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询会议录制列表。
func (l *ListMeetingRecordingsLogic) ListMeetingRecordings(in *live.ListMeetingRecordingsReq) (*live.ListMeetingRecordingsRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo); err != nil {
		return nil, meetingErr(err)
	}
	recs, total, err := l.svcCtx.MeetingRepo.ListRecordings(l.ctx, in.MeetingNo, in.Page, in.PageSize)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询录制列表失败")
	}
	out := make([]*live.MeetingRecordingInfo, 0, len(recs))
	for i := range recs {
		out = append(out, toRecordingInfo(&recs[i], l.svcCtx.Config.LiveKit.Record.PlayURLBase))
	}
	return &live.ListMeetingRecordingsRes{Recordings: out, Total: total}, nil
}
