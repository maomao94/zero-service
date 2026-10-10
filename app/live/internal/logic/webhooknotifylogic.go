package logic

import (
	"context"
	"strings"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

type WebhookNotifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWebhookNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WebhookNotifyLogic {
	return &WebhookNotifyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 接收 LiveKit webhook 事件（由 livegtw 验签后转发原始 proto 字节）。
// 逻辑内把字节解析为 livekit.WebhookEvent，基于 SDK 对象处理业务。
//
// 不做 TTL 幂等：事件可能重复、迟到、补发（LiveKit 推送/业务对账），
// 而本服务的处理操作本身幂等（参会记录 upsert、会议状态仅 active→ended
// 流转、unknown 会议安全忽略），重复/补发重放结果一致，保证业务闭环。
func (l *WebhookNotifyLogic) WebhookNotify(in *live.WebhookNotifyReq) (*live.WebhookNotifyRes, error) {
	if len(in.Data) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "webhook 数据不能为空")
	}
	event := &livekit.WebhookEvent{}
	if err := proto.Unmarshal(in.Data, event); err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "webhook 数据解析失败")
	}
	if strings.TrimSpace(event.GetId()) == "" || strings.TrimSpace(event.GetEvent()) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "webhook 事件缺失 id 或类型")
	}

	// 入口摘要：串起 hook 链路（事件 id/类型/房间/egress），便于按 id 追踪整条事件
	l.Logger.Infof("[webhook] 收到事件: id=%s, type=%s, room=%s, egress=%s",
		event.GetId(), event.GetEvent(), event.GetRoom().GetName(), event.GetEgressInfo().GetEgressId())

	switch event.GetEvent() {
	case "room_finished":
		l.handleRoomFinished(event)
	case "participant_joined":
		l.handleParticipantJoined(event)
	case "participant_left":
		l.handleParticipantLeft(event)
	case "room_started":
		// SIP 外呼或 API 创建的房间可能没有 meeting 记录，补插
		l.handleRoomStarted(event)
	case "participant_connection_aborted":
		// TODO: 参与者连接中止（弱网/断连），后续可标记 left 或触发告警
		l.logUnhandled(event)
	case "track_published", "track_unpublished":
		// TODO: 轨道发布/取消（后续可统计与会者音视频开关状态）
		l.logUnhandled(event)
	case "egress_started", "egress_updated", "egress_ended":
		// Egress 事件统一按"状态"分流（状态已 1:1 对齐 EgressStatus），不再按事件类型区分
		l.handleEgress(event)
	case "ingress_started", "ingress_ended":
		// TODO: 输入流事件（后续 Ingress 能力接入后同步状态）
		l.logUnhandled(event)
	default:
		// 未知事件安全忽略
		l.logUnhandled(event)
	}
	return &live.WebhookNotifyRes{}, nil
}

// logUnhandled 记录暂不处理的事件（含 TODO 项与未知事件）。
func (l *WebhookNotifyLogic) logUnhandled(event *livekit.WebhookEvent) {
	l.Logger.Infof("[webhook] 未处理事件: id=%s, type=%s, room=%s", event.GetId(), event.GetEvent(), event.GetRoom().GetName())
}

// handleRoomStarted 房间创建：已有记录则跳过（CreateMeeting/DialSipLogic 已创建）。
// 不再补插——SIP 外呼由 DialSipLogic 在拨号前创建会议，无需 webhook 补插。
func (l *WebhookNotifyLogic) handleRoomStarted(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	if room == nil || strings.TrimSpace(room.GetName()) == "" {
		l.Logger.Errorf("[webhook] room_started 缺少房间信息: id=%s", event.GetId())
		return
	}
	roomName := room.GetName()
	_, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, roomName)
	if err == nil {
		return // 已存在
	}
	l.Logger.Infof("[webhook] room_started 无对应会议记录（非托管房间属正常）: room=%s", roomName)
}

// handleRoomFinished 房间删除/会议结束：标记会议 ended（已结束则跳过）。
func (l *WebhookNotifyLogic) handleRoomFinished(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	if room == nil || strings.TrimSpace(room.GetName()) == "" {
		l.Logger.Errorf("[webhook] room_finished 缺少房间信息: id=%s", event.GetId())
		return
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, room.GetName())
	if err != nil {
		l.Logger.Infof("[webhook] room_finished 未知会议，跳过: room=%s", room.GetName())
		return
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return
	}
	if _, err := l.svcCtx.MeetingRepo.UpdateMeetingEnded(l.ctx, room.GetName(), carbonx.NowStartOfSecond().StdTime(), "", ""); err != nil {
		l.Logger.Errorf("[webhook] 更新会议结束失败: room=%s, err=%v", room.GetName(), err)
		return
	}
	l.Logger.Infof("[webhook] 会议已结束（webhook）: room=%s", room.GetName())
}

// handleParticipantJoined 参与者入会：upsert 参会记录（保留首次 join_time）。
func (l *WebhookNotifyLogic) handleParticipantJoined(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("[webhook] participant_joined 缺少房间/参与者信息: id=%s", event.GetId())
		return
	}
	p := &gormmodel.LiveMeetingParticipant{
		MeetingNo: room.GetName(),
		Identity:  participant.GetIdentity(),
		Name:      participant.GetName(),
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  carbonx.NowStartOfSecond().StdTime(),
	}
	if err := l.svcCtx.MeetingRepo.UpsertParticipant(l.ctx, p); err != nil {
		l.Logger.Errorf("[webhook] 参会记录写入失败: room=%s, identity=%s, err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("[webhook] 参与者入会: room=%s, identity=%s", room.GetName(), participant.GetIdentity())
}

// handleEgress 统一处理 Egress 事件（录制状态 1:1 对齐 EgressStatus）：
// 进行中（STARTING/ACTIVE/ENDING）upsert 记录；终态（COMPLETE/FAILED/ABORTED/LIMIT_REACHED）落库文件/错误。
// Egress 是 LiveKit 通用媒体输出（录制/推流/切片/截图），本服务只发起房间录制，
// 故仅处理带 room_name 的 Egress；其余安全忽略。
func (l *WebhookNotifyLogic) handleEgress(event *livekit.WebhookEvent) {
	info := event.GetEgressInfo()
	if info == nil || strings.TrimSpace(info.GetEgressId()) == "" {
		l.Logger.Errorf("[webhook] Egress 事件缺少 egress 信息: id=%s, type=%s", event.GetId(), event.GetEvent())
		return
	}
	if gormmodel.RecordingStatusIsTerminal(int(info.GetStatus())) {
		l.handleEgressTerminal(event, info)
		return
	}
	l.handleEgressActive(event, info)
}

// handleEgressActive 处理进行中的 Egress：按 egress_id 幂等 upsert 录制记录（状态取 Egress 状态）。
// 需要 room_name 才能建立录制记录，缺失则忽略。
func (l *WebhookNotifyLogic) handleEgressActive(event *livekit.WebhookEvent, info *livekit.EgressInfo) {
	status := int(info.GetStatus())
	if strings.TrimSpace(info.GetRoomName()) == "" {
		l.Logger.Infof("[webhook] 进行中的 Egress 缺少房间名，忽略: id=%s, egress=%s, status=%s", event.GetId(), info.GetEgressId(), egressStatusName(status))
		return
	}
	// 建立/刷新记录：start_time 先用 egress 顶层 started_at（受理时刻），真实媒体开始时间待 ACTIVE 同步
	recStart := carbonx.NowStartOfSecond().StdTime()
	if info.GetStartedAt() > 0 {
		recStart = time.Unix(0, info.GetStartedAt())
	}
	rec := &gormmodel.LiveMeetingRecording{
		MeetingNo: info.GetRoomName(),
		EgressId:  info.GetEgressId(),
		RoomName:  info.GetRoomName(),
		Status:    status,
		StartTime: recStart,
	}
	saved, err := l.svcCtx.MeetingRepo.SaveRecordingStarted(l.ctx, rec)
	if err != nil {
		l.Logger.Errorf("[webhook] 录制开始记录写入失败: id=%s, room=%s, egress=%s, status=%s, err=%v",
			event.GetId(), info.GetRoomName(), info.GetEgressId(), egressStatusName(status), err)
		return
	}
	// 以 Egress 为准覆盖真实录制开始时间（fileResults[].started_at，进入 ACTIVE 后才有）；值未变则跳过
	startedAt := saved.StartTime
	if realStart := egressStartedAt(info); !realStart.IsZero() && !realStart.Equal(startedAt) {
		if err := l.svcCtx.MeetingRepo.SyncRecordingStartTime(l.ctx, info.GetEgressId(), realStart); err != nil {
			l.Logger.Errorf("[webhook] 录制开始时间同步失败: id=%s, egress=%s, err=%v", event.GetId(), info.GetEgressId(), err)
		} else {
			startedAt = realStart
		}
	}
	l.Logger.Infof("[webhook] 录制进行中: id=%s, type=%s, room=%s, egress=%s, status=%s, startedAt=%s",
		event.GetId(), event.GetEvent(), info.GetRoomName(), info.GetEgressId(), egressStatusName(status), formatEgressTime(startedAt))
}

// handleEgressTerminal 处理终态的 Egress：仅 进行中 → 终态 落库（幂等），
// 已完成时提取文件信息（文件名/路径/大小/时长）。
func (l *WebhookNotifyLogic) handleEgressTerminal(event *livekit.WebhookEvent, info *livekit.EgressInfo) {
	status := int(info.GetStatus())
	result := egressFileResult(info, l.svcCtx.Config.LiveKit.Record.OutputDir)
	if info.GetStatus() != livekit.EgressStatus_EGRESS_COMPLETE {
		result.Error = info.GetError()
		if result.Error == "" {
			result.Error = info.GetDetails()
		}
		if result.Error == "" {
			result.Error = "egress ended with status " + info.GetStatus().String()
		}
	}
	updated, err := l.svcCtx.MeetingRepo.UpdateRecordingStatus(l.ctx, info.GetEgressId(), status, result)
	if err != nil {
		l.Logger.Errorf("[webhook] 录制状态更新失败: id=%s, room=%s, egress=%s, status=%s, err=%v",
			event.GetId(), info.GetRoomName(), info.GetEgressId(), egressStatusName(status), err)
		return
	}
	if !updated {
		// 可能已由 egress_updated 先标记完成但缺文件信息：仅为已完成记录补全文件字段
		if status == gormmodel.RecordingStatusComplete && result.FileName != "" {
			if err := l.svcCtx.MeetingRepo.BackfillRecordingFile(l.ctx, info.GetEgressId(), result.FileName, result.FilePath, result.FileSize, result.Duration, result.EndedAt); err != nil {
				l.Logger.Errorf("[webhook] 录制文件回填失败: id=%s, egress=%s, err=%v", event.GetId(), info.GetEgressId(), err)
			} else {
				l.Logger.Infof("[webhook] 录制文件已回填: id=%s, egress=%s, file=%s", event.GetId(), info.GetEgressId(), result.FileName)
			}
			return
		}
		// 记录不存在或已是终态：幂等忽略
		l.Logger.Infof("[webhook] Egress 结束事件被忽略（记录不存在或已是终态）: id=%s, egress=%s, status=%s",
			event.GetId(), info.GetEgressId(), egressStatusName(status))
		return
	}
	if status == gormmodel.RecordingStatusComplete {
		l.Logger.Infof("[webhook] 录制完成: id=%s, room=%s, egress=%s, duration=%ds, size=%dB, file=%s, endedAt=%s",
			event.GetId(), info.GetRoomName(), info.GetEgressId(), result.Duration, result.FileSize, result.FileName, formatEgressTime(result.EndedAt))
		return
	}
	l.Logger.Errorf("[webhook] 录制异常结束: id=%s, room=%s, egress=%s, status=%s, err=%s",
		event.GetId(), info.GetRoomName(), info.GetEgressId(), egressStatusName(status), result.Error)
}

// handleParticipantLeft 参与者离会：标记 left。
func (l *WebhookNotifyLogic) handleParticipantLeft(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("[webhook] participant_left 缺少房间/参与者信息: id=%s", event.GetId())
		return
	}
	if err := l.svcCtx.MeetingRepo.MarkParticipantLeft(l.ctx, room.GetName(), participant.GetIdentity(), carbonx.NowStartOfSecond().StdTime()); err != nil {
		l.Logger.Errorf("[webhook] 标记参与者离会失败: room=%s, identity=%s, err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("[webhook] 参与者离会: room=%s, identity=%s", room.GetName(), participant.GetIdentity())
}

// egressStatusName 返回 Egress 状态可读名（对齐 EgressStatus，如 EGRESS_ACTIVE）。
func egressStatusName(status int) string {
	return livekit.EgressStatus(status).String()
}

// formatEgressTime 用 carbon 格式化录制时间用于日志；零值返回 "-"。
func formatEgressTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return carbonx.FormatDateTime(t)
}
