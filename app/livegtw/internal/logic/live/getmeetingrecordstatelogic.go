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

type GetMeetingRecordStateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询会议当前录制状态
func NewGetMeetingRecordStateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingRecordStateLogic {
	return &GetMeetingRecordStateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeetingRecordStateLogic) GetMeetingRecordState(req *types.GetMeetingRecordStateRequest) (resp *types.GetMeetingRecordStateReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.GetMeetingRecordState(l.ctx, &live.GetMeetingRecordStateReq{
		MeetingNo: req.MeetingNo,
	})
	if err != nil {
		return nil, err
	}
	return &types.GetMeetingRecordStateReply{
		Recording: toRecordingInfo(r.GetRecording()),
		Active:    r.GetActive(),
	}, nil
}
