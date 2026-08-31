package livekitx

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

// RealtimeRoom 是一个参与者连接的轻量生命周期句柄，底层 Room 仍是 SDK 原生对象。
type RealtimeRoom struct {
	client      *Client
	room        *lksdk.Room
	ctx         context.Context    // Hook 分发使用的连接级 context，断开/关闭时取消
	cancel      context.CancelFunc // 取消 ctx
	once        sync.Once          // Close 幂等
	cleanupOnce sync.Once          // 连接索引清理幂等
}

// Connect 使用参与者 token 建立实时连接；token 的授权范围由业务决定。
//
// livekitx 内部构造默认回调桥：SDK 的 Room/Participant/Track/Data 回调会被
// 分发到对应 typed Hook（OnRoomConnection/OnParticipantConnected/OnTrack/...）。
// callback 中非 nil 字段按 SDK Merge 语义覆盖内置桥接的相同字段；全部字段为
// nil（或传 nil）时完整保留内置桥接。连接建立成功后合成分发一次
// RoomConnectionEvent{Status: Connected}。
func (c *Client) Connect(ctx context.Context, token string, callback *lksdk.RoomCallback, opts ...lksdk.ConnectOption) (*RealtimeRoom, error) {
	if c == nil || c.isClosed() {
		return nil, ErrClosed
	}
	if token == "" {
		return nil, ErrInvalidTokenOptions
	}
	if ctx == nil {
		return nil, ErrInvalidTokenOptions
	}
	r := &RealtimeRoom{client: c}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	room := lksdk.NewRoom(c.roomCallback(r, callback))
	r.room = room
	if err := room.JoinWithContextAndToken(ctx, c.config.URL, token, opts...); err != nil {
		r.cancel()
		return nil, err
	}
	state := ConnectionState{
		RoomName: room.Name(), Identity: room.LocalParticipant.Identity(),
		SessionID: uuid.NewString(), Status: ConnectionStatusConnected, UpdatedAt: time.Now(),
	}
	if err := c.store.Put(ctx, state); err != nil {
		room.Disconnect()
		r.cancel()
		return nil, err
	}
	c.stateMu.Lock()
	if c.closed {
		c.stateMu.Unlock()
		room.Disconnect()
		r.cancel()
		// 连接未登记到 realtime 索引，cleanup 不会清理，需直接删除 Store 记录，
		// 避免 Close 与 Connect 竞争时留下陈旧的活动连接索引。
		_ = c.store.Delete(context.Background(), state.SessionID)
		return nil, ErrClosed
	}
	c.realtime.Store(r, state.SessionID)
	c.stateMu.Unlock()
	_ = c.hooks.roomConnection.dispatch(r.ctx, RoomConnectionEvent{
		RoomName: room.Name(),
		Status:   RoomConnectionConnected,
	})
	return r, nil
}

// Room 返回 SDK 原生实时 Room，调用方只应通过 RealtimeRoom.Close 结束连接。
func (r *RealtimeRoom) Room() *lksdk.Room {
	if r == nil {
		return nil
	}
	return r.room
}

// Close 幂等断开实时连接，取消 Hook 分发 context，并从 Client 的资源集合中移除。
func (r *RealtimeRoom) Close() error {
	if r == nil {
		return nil
	}
	r.once.Do(func() {
		if r.room != nil {
			r.room.Disconnect()
		}
		r.cleanup()
	})
	return nil
}

// roomName 返回当前房间名；连接未建立或句柄为空时返回空字符串。
func (r *RealtimeRoom) roomName() string {
	if r == nil || r.room == nil {
		return ""
	}
	return r.room.Name()
}

// disconnectReason 返回 SDK 折叠原因与原始协议原因；房间未建立时为空值。
func (r *RealtimeRoom) disconnectReason() (lksdk.DisconnectionReason, livekit.DisconnectReason) {
	if r == nil || r.room == nil {
		return "", 0
	}
	return lksdk.GetDisconnectionReason(r.room.DisconnectReason()), r.room.DisconnectReason()
}

// cleanup 取消 Hook 分发 context 并从 Client 的活动连接索引中移除本连接。
// 由 Close 和 SDK 断开回调调用，幂等且并发安全。
func (r *RealtimeRoom) cleanup() {
	if r == nil {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	if r.client == nil {
		return
	}
	r.cleanupOnce.Do(func() {
		if value, ok := r.client.realtime.LoadAndDelete(r); ok {
			_ = r.client.store.Delete(context.Background(), value.(string))
		}
	})
}

// roomCallback 构造实时回调桥：内置桥接把 SDK 回调分发到 typed Hook，
// custom 中非 nil 字段按 SDK Merge 语义覆盖内置桥接字段。
func (c *Client) roomCallback(r *RealtimeRoom, custom *lksdk.RoomCallback) *lksdk.RoomCallback {
	cb := lksdk.NewRoomCallback()

	cb.OnDisconnected = func() {
		// SDK 同时触发 OnDisconnected 与 OnDisconnectedWithReason，
		// 只在 OnDisconnected 分发一次，原因从 room.DisconnectReason() 读取。
		reason, protocolReason := r.disconnectReason()
		_ = c.hooks.roomConnection.dispatch(r.ctx, RoomConnectionEvent{
			RoomName:       r.roomName(),
			Status:         RoomConnectionDisconnected,
			Reason:         reason,
			ProtocolReason: protocolReason,
		})
		r.cleanup()
	}
	cb.OnParticipantConnected = func(participant *lksdk.RemoteParticipant) {
		_ = c.hooks.participantConnected.dispatch(r.ctx, ParticipantConnectedEvent{
			RoomName: r.roomName(), Participant: participant,
		})
	}
	cb.OnParticipantDisconnected = func(participant *lksdk.RemoteParticipant) {
		_ = c.hooks.participantDisconnected.dispatch(r.ctx, ParticipantDisconnectedEvent{
			RoomName: r.roomName(), Participant: participant,
		})
	}
	cb.OnActiveSpeakersChanged = func(participants []lksdk.Participant) {
		_ = c.hooks.activeSpeakers.dispatch(r.ctx, ActiveSpeakersEvent{
			RoomName: r.roomName(), Participants: participants,
		})
	}
	cb.OnRoomMetadataChanged = func(metadata string) {
		_ = c.hooks.roomMetadata.dispatch(r.ctx, RoomMetadataEvent{
			RoomName: r.roomName(), Metadata: metadata,
		})
	}
	cb.OnRecordingStatusChanged = func(isRecording bool) {
		_ = c.hooks.roomRecording.dispatch(r.ctx, RoomRecordingEvent{
			RoomName: r.roomName(), IsRecording: isRecording,
		})
	}
	cb.OnRoomMovedWithSID = func(roomName, roomSID, token string) {
		// SDK 同时触发 OnRoomMoved 与 OnRoomMovedWithSID，只分发带 SID 的一次。
		_ = c.hooks.roomMoved.dispatch(r.ctx, RoomMovedEvent{
			RoomName: roomName, RoomSID: roomSID, Token: token,
		})
	}
	cb.OnReconnecting = func() {
		_ = c.hooks.roomConnection.dispatch(r.ctx, RoomConnectionEvent{
			RoomName: r.roomName(), Status: RoomConnectionReconnecting,
		})
	}
	cb.OnReconnected = func() {
		_ = c.hooks.roomConnection.dispatch(r.ctx, RoomConnectionEvent{
			RoomName: r.roomName(), Status: RoomConnectionReconnected,
		})
	}
	cb.OnLocalTrackSubscribed = func(publication *lksdk.LocalTrackPublication, lp *lksdk.LocalParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventLocalSubscribed,
			Publication: publication, Local: lp,
		})
	}
	cb.OnLocalTrackPublished = func(publication *lksdk.LocalTrackPublication, lp *lksdk.LocalParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventLocalPublished,
			Publication: publication, Local: lp,
		})
	}
	cb.OnLocalTrackUnpublished = func(publication *lksdk.LocalTrackPublication, lp *lksdk.LocalParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventLocalUnpublished,
			Publication: publication, Local: lp,
		})
	}
	cb.OnTrackMuted = func(pub lksdk.TrackPublication, p lksdk.Participant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventMuted, Publication: pub, Participant: p,
		})
	}
	cb.OnTrackUnmuted = func(pub lksdk.TrackPublication, p lksdk.Participant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventUnmuted, Publication: pub, Participant: p,
		})
	}
	cb.OnMetadataChanged = func(oldMetadata string, p lksdk.Participant) {
		_ = c.hooks.participantMetadata.dispatch(r.ctx, ParticipantMetadataEvent{
			RoomName: r.roomName(), Participant: p, OldMetadata: oldMetadata,
		})
	}
	cb.OnAttributesChanged = func(changed map[string]string, p lksdk.Participant) {
		_ = c.hooks.participantAttributes.dispatch(r.ctx, ParticipantAttributesEvent{
			RoomName: r.roomName(), Participant: p, Changed: changed,
		})
	}
	cb.OnIsSpeakingChanged = func(p lksdk.Participant) {
		_ = c.hooks.participantSpeaking.dispatch(r.ctx, ParticipantSpeakingEvent{
			RoomName: r.roomName(), Participant: p, IsSpeaking: p.IsSpeaking(),
		})
	}
	cb.OnConnectionQualityChanged = func(update *livekit.ConnectionQualityInfo, p lksdk.Participant) {
		_ = c.hooks.connectionQuality.dispatch(r.ctx, ConnectionQualityEvent{
			RoomName: r.roomName(), Participant: p, Update: update,
		})
	}
	cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventSubscribed,
			Track: track, Publication: publication, Participant: rp,
		})
	}
	cb.OnTrackUnsubscribed = func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventUnsubscribed,
			Track: track, Publication: publication, Participant: rp,
		})
	}
	cb.OnTrackSubscriptionFailed = func(sid string, rp *lksdk.RemoteParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventSubscriptionFailed,
			TrackSID: sid, Participant: rp,
		})
	}
	cb.OnTrackPublished = func(publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventPublished,
			Publication: publication, Participant: rp,
		})
	}
	cb.OnTrackUnpublished = func(publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		_ = c.hooks.track.dispatch(r.ctx, TrackEvent{
			RoomName: r.roomName(), Kind: TrackEventUnpublished,
			Publication: publication, Participant: rp,
		})
	}
	cb.OnDataPacket = func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
		c.bridgeDataPacket(r, data, params)
	}

	// custom 中非 nil 字段覆盖内置桥接；全 nil（或 nil）时完整保留内置桥接。
	cb.Merge(custom)
	return cb
}

// bridgeDataPacket 把 SDK Data 回调分发为 DataEvent，并按约定解析聊天消息
// 分发 ChatMessageEvent：*livekit.ChatMessage 直接识别为聊天；
// topic == ChatTopic 的 UserDataPacket 也识别为聊天。
func (c *Client) bridgeDataPacket(r *RealtimeRoom, data lksdk.DataPacket, params lksdk.DataReceiveParams) {
	topic, payload := dataTopicPayload(data)
	event := DataEvent{
		RoomName:       r.roomName(),
		SenderIdentity: params.SenderIdentity,
		Topic:          topic,
		Payload:        payload,
		Packet:         data,
	}
	if params.Sender != nil {
		event.SenderSID = params.Sender.SID()
	}
	_ = c.hooks.data.dispatch(r.ctx, event)

	switch msg := data.(type) {
	case *livekit.ChatMessage:
		_ = c.hooks.dispatchChatMessage(r.ctx, ChatMessageEvent{
			RoomName:  r.roomName(),
			SenderID:  params.SenderIdentity,
			MessageID: msg.GetId(),
			Topic:     ChatTopic,
			Text:      msg.GetMessage(),
			Timestamp: msg.GetTimestamp(),
			Deleted:   msg.GetDeleted(),
			Generated: msg.GetGenerated(),
		})
	case *lksdk.UserDataPacket:
		if msg.Topic == ChatTopic {
			_ = c.hooks.dispatchChatMessage(r.ctx, ChatMessageEvent{
				RoomName: r.roomName(),
				SenderID: params.SenderIdentity,
				Topic:    msg.Topic,
				Text:     string(msg.Payload),
				Payload:  msg.Payload,
			})
		}
	}
}

// dataTopicPayload 从任意 DataPacket 中提取 topic 与 payload 用于 DataEvent。
func dataTopicPayload(data lksdk.DataPacket) (topic string, payload []byte) {
	switch msg := data.(type) {
	case *lksdk.UserDataPacket:
		return msg.Topic, msg.Payload
	case *livekit.ChatMessage:
		return ChatTopic, []byte(msg.GetMessage())
	default:
		if proto := data.ToProto(); proto != nil {
			if user := proto.GetUser(); user != nil {
				return user.GetTopic(), user.GetPayload()
			}
			if chat := proto.GetChatMessage(); chat != nil {
				return ChatTopic, []byte(chat.GetMessage())
			}
		}
		return "", nil
	}
}
