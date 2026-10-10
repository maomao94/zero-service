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
	case "egress_started":
		l.handleEgressStarted(event)
	case "egress_updated":
		l.handleEgressUpdated(event)
	case "egress_ended":
		l.handleEgressEnded(event)
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
	l.Logger.Infof("[webhook] webhook event unhandled: id=%s type=%s room=%s", event.GetId(), event.GetEvent(), event.GetRoom().GetName())
}

// handleRoomStarted 房间创建：已有记录则跳过（CreateMeeting/DialSipLogic 已创建）。
// 不再补插——SIP 外呼由 DialSipLogic 在拨号前创建会议，无需 webhook 补插。
func (l *WebhookNotifyLogic) handleRoomStarted(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	if room == nil || strings.TrimSpace(room.GetName()) == "" {
		l.Logger.Errorf("[webhook] room_started without room: id=%s", event.GetId())
		return
	}
	roomName := room.GetName()
	_, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, roomName)
	if err == nil {
		return // 已存在
	}
	l.Logger.Infof("[webhook] room_started without meeting record (expected for non-managed rooms): room=%s", roomName)
}

// handleRoomFinished 房间删除/会议结束：标记会议 ended（已结束则跳过）。
func (l *WebhookNotifyLogic) handleRoomFinished(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	if room == nil || strings.TrimSpace(room.GetName()) == "" {
		l.Logger.Errorf("[webhook] room_finished without room: id=%s", event.GetId())
		return
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, room.GetName())
	if err != nil {
		l.Logger.Infof("[webhook] room_finished for unknown meeting, skip: room=%s", room.GetName())
		return
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return
	}
	if _, err := l.svcCtx.MeetingRepo.UpdateMeetingEnded(l.ctx, room.GetName(), carbonx.NowStartOfSecond().StdTime(), "", ""); err != nil {
		l.Logger.Errorf("[webhook] update meeting ended failed: room=%s err=%v", room.GetName(), err)
		return
	}
	l.Logger.Infof("[webhook] meeting ended by webhook: %s", room.GetName())
}

// handleParticipantJoined 参与者入会：upsert 参会记录（保留首次 join_time）。
func (l *WebhookNotifyLogic) handleParticipantJoined(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("[webhook] participant_joined without room/participant: id=%s", event.GetId())
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
		l.Logger.Errorf("[webhook] upsert participant failed: room=%s identity=%s err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("[webhook] participant joined: room=%s identity=%s", room.GetName(), participant.GetIdentity())
}

// handleEgressStarted Egress 开始/进行中：按 egress_id 幂等 upsert 录制记录（status 取 Egress 状态）。
// 只处理会议房间的 Egress（room_name 非空）；其它类型安全忽略。
func (l *WebhookNotifyLogic) handleEgressStarted(event *livekit.WebhookEvent) {
	info := event.GetEgressInfo()
	if info == nil || strings.TrimSpace(info.GetEgressId()) == "" || strings.TrimSpace(info.GetRoomName()) == "" {
		l.logUnhandled(event)
		return
	}
	startedAt := carbonx.NowStartOfSecond().StdTime()
	if info.GetStartedAt() > 0 {
		startedAt = time.Unix(0, info.GetStartedAt())
	}
	rec := &gormmodel.LiveMeetingRecording{
		MeetingNo: info.GetRoomName(),
		EgressId:  info.GetEgressId(),
		RoomName:  info.GetRoomName(),
		Status:    int(info.GetStatus()),
		StartTime: startedAt,
	}
	if _, err := l.svcCtx.MeetingRepo.SaveRecordingStarted(l.ctx, rec); err != nil {
		l.Logger.Errorf("[webhook] upsert recording failed: egress=%s room=%s err=%v", info.GetEgressId(), info.GetRoomName(), err)
		return
	}
	l.Logger.Infof("[webhook] egress started: room=%s egress=%s", info.GetRoomName(), info.GetEgressId())
}

// handleEgressUpdated Egress 状态更新：终态转 handleEgressEnded，否则按开始处理。
func (l *WebhookNotifyLogic) handleEgressUpdated(event *livekit.WebhookEvent) {
	info := event.GetEgressInfo()
	if info == nil {
		l.logUnhandled(event)
		return
	}
	switch info.GetStatus() {
	case livekit.EgressStatus_EGRESS_COMPLETE, livekit.EgressStatus_EGRESS_FAILED,
		livekit.EgressStatus_EGRESS_ABORTED, livekit.EgressStatus_EGRESS_LIMIT_REACHED:
		l.handleEgressEnded(event)
	default:
		l.handleEgressStarted(event)
	}
}

// handleEgressEnded Egress 结束：终态落库（仅 进行中 → 终态，幂等），
// 状态直接取 Egress 状态（COMPLETE/FAILED/ABORTED/LIMIT_REACHED），
// 已完成时提取文件信息（文件名/路径/大小/时长）。
func (l *WebhookNotifyLogic) handleEgressEnded(event *livekit.WebhookEvent) {
	info := event.GetEgressInfo()
	if info == nil || strings.TrimSpace(info.GetEgressId()) == "" {
		l.Logger.Errorf("[webhook] egress_ended without egress info: id=%s", event.GetId())
		return
	}
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
		l.Logger.Errorf("[webhook] update recording status failed: egress=%s err=%v", info.GetEgressId(), err)
		return
	}
	if !updated {
		// 可能已由 egress_updated 先标记完成但缺文件信息：仅为已完成记录补全文件字段
		if status == gormmodel.RecordingStatusComplete && result.FileName != "" {
			if err := l.svcCtx.MeetingRepo.BackfillRecordingFile(l.ctx, info.GetEgressId(), result.FileName, result.FilePath, result.FileSize, result.Duration, result.EndedAt); err != nil {
				l.Logger.Errorf("[webhook] backfill recording file failed: egress=%s err=%v", info.GetEgressId(), err)
			} else {
				l.Logger.Infof("[webhook] recording file backfilled: egress=%s file=%s", info.GetEgressId(), result.FileName)
			}
			return
		}
		// 记录不存在或已是终态：幂等忽略
		l.Logger.Infof("[webhook] egress ended ignored (no recording or already terminal): egress=%s status=%s", info.GetEgressId(), info.GetStatus().String())
		return
	}
	l.Logger.Infof("[webhook] egress ended: room=%s egress=%s status=%d file=%s", info.GetRoomName(), info.GetEgressId(), status, result.FileName)
}

// handleParticipantLeft 参与者离会：标记 left。
func (l *WebhookNotifyLogic) handleParticipantLeft(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("[webhook] participant_left without room/participant: id=%s", event.GetId())
		return
	}
	if err := l.svcCtx.MeetingRepo.MarkParticipantLeft(l.ctx, room.GetName(), participant.GetIdentity(), carbonx.NowStartOfSecond().StdTime()); err != nil {
		l.Logger.Errorf("[webhook] mark participant left failed: room=%s identity=%s err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("[webhook] participant left: room=%s identity=%s", room.GetName(), participant.GetIdentity())
}
