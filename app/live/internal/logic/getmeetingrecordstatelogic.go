package logic

import (
	"context"
	"errors"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeetingRecordStateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMeetingRecordStateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingRecordStateLogic {
	return &GetMeetingRecordStateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询会议当前录制状态：返回进行中的录制（若有）与 LiveKit 侧是否活跃。
// 前端据此渲染录制按钮，无需扫描录制列表猜测状态。
func (l *GetMeetingRecordStateLogic) GetMeetingRecordState(in *live.GetMeetingRecordStateReq) (*live.GetMeetingRecordStateRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo); err != nil {
		return nil, meetingErr(err)
	}
	rec, err := l.svcCtx.MeetingRepo.GetActiveRecordingByMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		if errors.Is(err, svc.ErrRecordingNotFound) {
			return &live.GetMeetingRecordStateRes{Active: false}, nil
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询录制记录失败")
	}
	active, rerr := reconcileRecordingState(l.ctx, l.svcCtx, rec)
	if rerr != nil {
		// 对账失败：按 DB 状态判断，保守上报"进行中"
		l.Logger.Errorf("reconcile record state failed: egress=%s err=%v", rec.EgressId, rerr)
		return &live.GetMeetingRecordStateRes{Recording: toRecordingInfo(rec, l.svcCtx.Config.LiveKit.Record.PlayURLBase), Active: true}, nil
	}
	if active {
		return &live.GetMeetingRecordStateRes{Recording: toRecordingInfo(rec, l.svcCtx.Config.LiveKit.Record.PlayURLBase), Active: true}, nil
	}
	// 已被对账落终态：无进行中
	return &live.GetMeetingRecordStateRes{Active: false}, nil
}
