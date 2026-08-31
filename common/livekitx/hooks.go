package livekitx

import (
	"context"
	"errors"
	"slices"
	"sync"
)

// Handler 处理一个 typed 事件；同一事件的 handler 按注册顺序快照执行，
// 返回的错误会被聚合后返回给分发调用方（Webhook 接收方或桥接调用方）。
type Handler[T any] func(context.Context, T) error

// ChatMessageHandler 处理一个聊天消息事件。
type ChatMessageHandler = Handler[ChatMessageEvent]

// WebhookHandler 在签名校验成功后同步执行；重复事件仍交给业务按 ID 去重。
type WebhookHandler = Handler[*WebhookEvent]

// Subscription 表示一个可注销的 Hook 订阅；Unsubscribe 幂等且并发安全。
type Subscription interface{ Unsubscribe() }

// eventDispatcher 按注册顺序快照分发 typed 事件。
//
// 分发契约：
//   - 同一事件类型的 handler 按注册顺序执行；注册/注销不会阻塞已开始的 handler。
//   - 默认同步执行，handler 返回的错误被聚合（errors.Join）返回。
//   - handler panic 会被 recover 并转换为 *HookPanicError，不会穿透 SDK 读循环。
//   - 关闭后 dispatch 返回 ErrClosed；关闭后 add 返回空订阅。
type eventDispatcher[T any] struct {
	mu       sync.RWMutex
	closed   bool
	nextID   uint64
	event    string // 事件类型名，用于 panic 错误和日志
	handlers map[uint64]Handler[T]
}

// newEventDispatcher 创建按注册顺序分发的事件 dispatcher。
func newEventDispatcher[T any](event string) *eventDispatcher[T] {
	return &eventDispatcher[T]{event: event, handlers: make(map[uint64]Handler[T])}
}

// add 注册 handler 并返回注销句柄；dispatcher 已关闭或 handler 为 nil 时返回空订阅。
func (d *eventDispatcher[T]) add(handler Handler[T]) Subscription {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || handler == nil {
		return &subscription{}
	}
	d.nextID++
	id := d.nextID
	d.handlers[id] = handler
	return &subscription{unsubscribe: func() {
		d.mu.Lock()
		delete(d.handlers, id)
		d.mu.Unlock()
	}}
}

// dispatch 按注册顺序快照并同步执行全部 handler，聚合所有错误后返回。
func (d *eventDispatcher[T]) dispatch(ctx context.Context, event T) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrClosed
	}
	// map 迭代顺序不稳定，先按注册 ID 排序再快照，保证注册顺序执行。
	ids := make([]uint64, 0, len(d.handlers))
	for id := range d.handlers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	handlers := make([]Handler[T], 0, len(ids))
	for _, id := range ids {
		handlers = append(handlers, d.handlers[id])
	}
	d.mu.RUnlock()

	var errs []error
	for _, handler := range handlers {
		if err := callHandler(ctx, d.event, handler, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// close 标记 dispatcher 关闭并释放 handler 表；已开始的 handler 不受影响。
func (d *eventDispatcher[T]) close() {
	d.mu.Lock()
	d.closed = true
	d.handlers = nil
	d.mu.Unlock()
}

// callHandler 带 panic recovery 调用单个 handler，panic 转换为 *HookPanicError。
func callHandler[T any](ctx context.Context, event string, handler Handler[T], value T) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &HookPanicError{Event: event, Value: recovered}
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	return handler(ctx, value)
}

// subscription 是幂等且并发安全的注销句柄。
type subscription struct {
	once        sync.Once
	unsubscribe func()
}

func (s *subscription) Unsubscribe() {
	if s == nil || s.unsubscribe == nil {
		return
	}
	s.once.Do(s.unsubscribe)
}

// hookSet 持有全部 typed Hook dispatcher；每个事件类型一个实例。
type hookSet struct {
	roomConnection          *eventDispatcher[RoomConnectionEvent]
	roomMetadata            *eventDispatcher[RoomMetadataEvent]
	roomRecording           *eventDispatcher[RoomRecordingEvent]
	roomMoved               *eventDispatcher[RoomMovedEvent]
	activeSpeakers          *eventDispatcher[ActiveSpeakersEvent]
	participantConnected    *eventDispatcher[ParticipantConnectedEvent]
	participantDisconnected *eventDispatcher[ParticipantDisconnectedEvent]
	participantMetadata     *eventDispatcher[ParticipantMetadataEvent]
	participantAttributes   *eventDispatcher[ParticipantAttributesEvent]
	participantSpeaking     *eventDispatcher[ParticipantSpeakingEvent]
	connectionQuality       *eventDispatcher[ConnectionQualityEvent]
	track                   *eventDispatcher[TrackEvent]
	data                    *eventDispatcher[DataEvent]
	chat                    *eventDispatcher[ChatMessageEvent]
	rpc                     *eventDispatcher[RPCRequestEvent]
	webhook                 *eventDispatcher[*WebhookEvent]
}

// newHookSet 创建包含全部事件 dispatcher 的 Hook 集合。
func newHookSet() *hookSet {
	return &hookSet{
		roomConnection:          newEventDispatcher[RoomConnectionEvent]("room connection"),
		roomMetadata:            newEventDispatcher[RoomMetadataEvent]("room metadata"),
		roomRecording:           newEventDispatcher[RoomRecordingEvent]("room recording"),
		roomMoved:               newEventDispatcher[RoomMovedEvent]("room moved"),
		activeSpeakers:          newEventDispatcher[ActiveSpeakersEvent]("active speakers"),
		participantConnected:    newEventDispatcher[ParticipantConnectedEvent]("participant connected"),
		participantDisconnected: newEventDispatcher[ParticipantDisconnectedEvent]("participant disconnected"),
		participantMetadata:     newEventDispatcher[ParticipantMetadataEvent]("participant metadata"),
		participantAttributes:   newEventDispatcher[ParticipantAttributesEvent]("participant attributes"),
		participantSpeaking:     newEventDispatcher[ParticipantSpeakingEvent]("participant speaking"),
		connectionQuality:       newEventDispatcher[ConnectionQualityEvent]("connection quality"),
		track:                   newEventDispatcher[TrackEvent]("track"),
		data:                    newEventDispatcher[DataEvent]("data"),
		chat:                    newEventDispatcher[ChatMessageEvent]("chat message"),
		rpc:                     newEventDispatcher[RPCRequestEvent]("rpc request"),
		webhook:                 newEventDispatcher[*WebhookEvent]("webhook"),
	}
}

// close 关闭全部 dispatcher；关闭后不再执行任何 handler。
func (s *hookSet) close() {
	if s == nil {
		return
	}
	s.roomConnection.close()
	s.roomMetadata.close()
	s.roomRecording.close()
	s.roomMoved.close()
	s.activeSpeakers.close()
	s.participantConnected.close()
	s.participantDisconnected.close()
	s.participantMetadata.close()
	s.participantAttributes.close()
	s.participantSpeaking.close()
	s.connectionQuality.close()
	s.track.close()
	s.data.close()
	s.chat.close()
	s.rpc.close()
	s.webhook.close()
}

// onChatMessage 注册聊天消息 handler（供桥接与测试内部使用）。
func (s *hookSet) onChatMessage(handler ChatMessageHandler) Subscription {
	return s.chat.add(handler)
}

// dispatchChatMessage 分发聊天消息；由 SDK Data 回调桥接真实触发。
func (s *hookSet) dispatchChatMessage(ctx context.Context, event ChatMessageEvent) error {
	return s.chat.dispatch(ctx, event)
}

// OnRoomConnection 注册连接状态变化 Hook（入会成功/重连/断开），用于会议状态同步。
func (c *Client) OnRoomConnection(handler Handler[RoomConnectionEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.roomConnection.add(handler)
}

// OnRoomMetadata 注册房间 metadata 变化 Hook。
func (c *Client) OnRoomMetadata(handler Handler[RoomMetadataEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.roomMetadata.add(handler)
}

// OnRoomRecording 注册录制状态变化 Hook。
func (c *Client) OnRoomRecording(handler Handler[RoomRecordingEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.roomRecording.add(handler)
}

// OnRoomMoved 注册房间迁移 Hook。
func (c *Client) OnRoomMoved(handler Handler[RoomMovedEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.roomMoved.add(handler)
}

// OnActiveSpeakers 注册活跃说话人变化 Hook。
func (c *Client) OnActiveSpeakers(handler Handler[ActiveSpeakersEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.activeSpeakers.add(handler)
}

// OnParticipantConnected 注册远端参与者入会 Hook，用于参与者变更同步。
func (c *Client) OnParticipantConnected(handler Handler[ParticipantConnectedEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.participantConnected.add(handler)
}

// OnParticipantDisconnected 注册远端参与者离会 Hook，用于参与者变更同步。
func (c *Client) OnParticipantDisconnected(handler Handler[ParticipantDisconnectedEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.participantDisconnected.add(handler)
}

// OnParticipantMetadata 注册参与者 metadata 变化 Hook。
func (c *Client) OnParticipantMetadata(handler Handler[ParticipantMetadataEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.participantMetadata.add(handler)
}

// OnParticipantAttributes 注册参与者属性变化 Hook。
func (c *Client) OnParticipantAttributes(handler Handler[ParticipantAttributesEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.participantAttributes.add(handler)
}

// OnParticipantSpeaking 注册参与者说话状态变化 Hook。
func (c *Client) OnParticipantSpeaking(handler Handler[ParticipantSpeakingEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.participantSpeaking.add(handler)
}

// OnConnectionQuality 注册参与者连接质量变化 Hook。
func (c *Client) OnConnectionQuality(handler Handler[ConnectionQualityEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.connectionQuality.add(handler)
}

// OnTrack 注册轨道状态变化 Hook（发布/订阅/静音/失败等），用于轨道状态同步。
func (c *Client) OnTrack(handler Handler[TrackEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.track.add(handler)
}

// OnData 注册任意 DataPacket 接收 Hook，用于数据链路审计或业务数据分发。
func (c *Client) OnData(handler Handler[DataEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.data.add(handler)
}

// OnChatMessage 注册聊天消息 Hook；消息持久化、历史、未读数和审核由业务负责。
func (c *Client) OnChatMessage(handler ChatMessageHandler) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.onChatMessage(handler)
}

// OnRPCRequest 注册 RPC 请求 Hook（仅覆盖通过 livekitx.RegisterRPC 注册的方法）。
func (c *Client) OnRPCRequest(handler Handler[RPCRequestEvent]) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.rpc.add(handler)
}

// OnWebhook 注册 Webhook Hook；ReceiveWebhook 验签成功后按注册顺序分发。
func (c *Client) OnWebhook(handler WebhookHandler) Subscription {
	if c == nil || c.hooks == nil {
		return &subscription{}
	}
	return c.hooks.webhook.add(handler)
}
