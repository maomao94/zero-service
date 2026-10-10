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

type StopMeetingRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 停止会议录制
func NewStopMeetingRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopMeetingRecordLogic {
	return &StopMeetingRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StopMeetingRecordLogic) StopMeetingRecord(req *types.StopMeetingRecordRequest) (resp *types.StopMeetingRecordReply, err error) {
	if _, err = l.svcCtx.LiveRpcCli.StopMeetingRecord(l.ctx, &live.StopMeetingRecordReq{
		MeetingNo: req.MeetingNo,
		RecordId:  req.RecordId,
	}); err != nil {
		return nil, err
	}
	return &types.StopMeetingRecordReply{}, nil
}
