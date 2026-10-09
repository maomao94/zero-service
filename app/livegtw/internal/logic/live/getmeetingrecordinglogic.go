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

type GetMeetingRecordingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询录制详情
func NewGetMeetingRecordingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingRecordingLogic {
	return &GetMeetingRecordingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeetingRecordingLogic) GetMeetingRecording(req *types.GetMeetingRecordingRequest) (resp *types.GetMeetingRecordingReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.GetMeetingRecording(l.ctx, &live.GetMeetingRecordingReq{
		RecordId: req.RecordId,
	})
	if err != nil {
		return nil, err
	}
	return &types.GetMeetingRecordingReply{Recording: toRecordingInfo(r.GetRecording())}, nil
}
