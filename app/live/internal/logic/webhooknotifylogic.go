package logic

import (
	"context"
	"strings"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
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
		// 房间创建即 active（CreateMeeting 已置状态），无需处理
	case "participant_connection_aborted":
		// TODO: 参与者连接中止（弱网/断连），后续可标记 left 或触发告警
		l.logUnhandled(event)
	case "track_published", "track_unpublished":
		// TODO: 轨道发布/取消（后续可统计与会者音视频开关状态）
		l.logUnhandled(event)
	case "egress_started", "egress_updated", "egress_ended":
		// TODO: 录制事件（后续录制能力接入后同步录制状态）
		l.logUnhandled(event)
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
	l.Logger.Infof("webhook event unhandled: id=%s type=%s room=%s", event.GetId(), event.GetEvent(), event.GetRoom().GetName())
}

// handleRoomFinished 房间删除/会议结束：标记会议 ended（已结束则跳过）。
func (l *WebhookNotifyLogic) handleRoomFinished(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	if room == nil || strings.TrimSpace(room.GetName()) == "" {
		l.Logger.Errorf("room_finished without room: id=%s", event.GetId())
		return
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, room.GetName())
	if err != nil {
		l.Logger.Infof("room_finished for unknown meeting, skip: room=%s", room.GetName())
		return
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return
	}
	if _, err := l.svcCtx.MeetingRepo.UpdateMeetingEnded(l.ctx, room.GetName(), time.Now(), "", ""); err != nil {
		l.Logger.Errorf("update meeting ended failed: room=%s err=%v", room.GetName(), err)
		return
	}
	l.Logger.Infof("meeting ended by webhook: %s", room.GetName())
}

// handleParticipantJoined 参与者入会：upsert 参会记录（保留首次 join_time）。
func (l *WebhookNotifyLogic) handleParticipantJoined(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("participant_joined without room/participant: id=%s", event.GetId())
		return
	}
	p := &gormmodel.LiveMeetingParticipant{
		MeetingNo: room.GetName(),
		Identity:  participant.GetIdentity(),
		Name:      participant.GetName(),
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  time.Now(),
	}
	if err := l.svcCtx.MeetingRepo.UpsertParticipant(l.ctx, p); err != nil {
		l.Logger.Errorf("upsert participant failed: room=%s identity=%s err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("participant joined: room=%s identity=%s", room.GetName(), participant.GetIdentity())
}

// handleParticipantLeft 参与者离会：标记 left。
func (l *WebhookNotifyLogic) handleParticipantLeft(event *livekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil || strings.TrimSpace(room.GetName()) == "" || strings.TrimSpace(participant.GetIdentity()) == "" {
		l.Logger.Errorf("participant_left without room/participant: id=%s", event.GetId())
		return
	}
	if err := l.svcCtx.MeetingRepo.MarkParticipantLeft(l.ctx, room.GetName(), participant.GetIdentity(), time.Now()); err != nil {
		l.Logger.Errorf("mark participant left failed: room=%s identity=%s err=%v", room.GetName(), participant.GetIdentity(), err)
		return
	}
	l.Logger.Infof("participant left: room=%s identity=%s", room.GetName(), participant.GetIdentity())
}