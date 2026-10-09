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

type StartMeetingRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 开始会议录制
func NewStartMeetingRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartMeetingRecordLogic {
	return &StartMeetingRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartMeetingRecordLogic) StartMeetingRecord(req *types.StartMeetingRecordRequest) (resp *types.StartMeetingRecordReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.StartMeetingRecord(l.ctx, &live.StartMeetingRecordReq{
		MeetingNo: req.MeetingNo,
		AudioOnly: req.AudioOnly,
		Layout:    req.Layout,
	})
	if err != nil {
		return nil, err
	}
	return &types.StartMeetingRecordReply{Recording: toRecordingInfo(r.GetRecording())}, nil
}
