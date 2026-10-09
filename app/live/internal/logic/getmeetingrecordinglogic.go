package logic

import (
	"context"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeetingRecordingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMeetingRecordingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingRecordingLogic {
	return &GetMeetingRecordingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询单个录制详情。
func (l *GetMeetingRecordingLogic) GetMeetingRecording(in *live.GetMeetingRecordingReq) (*live.GetMeetingRecordingRes, error) {
	if strings.TrimSpace(in.RecordId) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "录制记录ID不能为空")
	}
	rec, err := l.svcCtx.MeetingRepo.GetRecording(l.ctx, in.RecordId)
	if err != nil {
		return nil, recordingErr(err)
	}
	return &live.GetMeetingRecordingRes{Recording: toRecordingInfo(rec, l.svcCtx.Config.LiveKit.Record.PlayURLBase)}, nil
}
