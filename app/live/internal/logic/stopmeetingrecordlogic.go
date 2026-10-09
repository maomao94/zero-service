package logic

import (
	"context"
	"errors"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
)

type StopMeetingRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopMeetingRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopMeetingRecordLogic {
	return &StopMeetingRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 停止会议录制：定位录制记录（record_id 优先，否则按会议号取"录制中"记录），
// 调 StopEgress 停止任务；最终状态由 webhook egress_ended 更新（幂等）。
func (l *StopMeetingRecordLogic) StopMeetingRecord(in *live.StopMeetingRecordReq) (*live.StopMeetingRecordRes, error) {
	rec, meeting, err := l.resolveRecording(in)
	if err != nil {
		return nil, err
	}
	// 会议无"录制中"记录：幂等成功（避免重复停止报错）
	if rec == nil {
		return &live.StopMeetingRecordRes{}, nil
	}
	// 仅会议创建者（主持人）可操作录制；系统创建放行
	if err := requireMeetingOperator(meeting, authctx.GetUserId(l.ctx)); err != nil {
		return nil, err
	}
	// 已进入终态：幂等成功返回
	if !gormmodel.RecordingStatusIsActive(rec.Status) {
		return &live.StopMeetingRecordRes{}, nil
	}
	if _, err := l.svcCtx.LiveKit.API().Egress().StopEgress(l.ctx, &livekit.StopEgressRequest{EgressId: rec.EgressId}); err != nil {
		// 停止失败可能是 egress 已结束/不存在：对账兜底，能落终态则视为已停止（幂等），避免 UI 卡在"录制中"
		active, rerr := reconcileRecordingState(l.ctx, l.svcCtx, rec)
		if rerr == nil && !active {
			l.Logger.Infof("meeting record already finalized: meeting=%s egress=%s", rec.MeetingNo, rec.EgressId)
			return &live.StopMeetingRecordRes{}, nil
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "停止录制失败")
	}
	l.Logger.Infof("meeting record stopping: meeting=%s egress=%s", rec.MeetingNo, rec.EgressId)
	return &live.StopMeetingRecordRes{}, nil
}

// resolveRecording 定位待停止的录制记录及所属会议。
// record_id 优先；否则按会议号取"录制中"记录（无则返回 nil, meeting, nil 表示幂等成功）。
func (l *StopMeetingRecordLogic) resolveRecording(in *live.StopMeetingRecordReq) (*gormmodel.LiveMeetingRecording, *gormmodel.LiveMeeting, error) {
	if strings.TrimSpace(in.RecordId) != "" {
		rec, err := l.svcCtx.MeetingRepo.GetRecording(l.ctx, in.RecordId)
		if err != nil {
			return nil, nil, recordingErr(err)
		}
		// 会议单据不存在（含软删）即拒绝：避免"找不到会议=跳过校验"被越权利用
		meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, rec.MeetingNo)
		if err != nil {
			return nil, nil, meetingErr(err)
		}
		return rec, meeting, nil
	}
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, nil, err
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, nil, meetingErr(err)
	}
	rec, err := l.svcCtx.MeetingRepo.GetActiveRecordingByMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		if errors.Is(err, svc.ErrRecordingNotFound) {
			return nil, meeting, nil
		}
		return nil, nil, recordingErr(err)
	}
	return rec, meeting, nil
}
