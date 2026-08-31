package livekitx

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// newBridgeTestClient 构造用于桥接单元测试的 Client 和未连接的 RealtimeRoom。
func newBridgeTestClient(t *testing.T) (*Client, *RealtimeRoom) {
	t.Helper()
	client, err := New(WithURL("http://127.0.0.1:7880"), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	room := &RealtimeRoom{client: client}
	room.ctx, room.cancel = context.WithCancel(context.Background())
	// 未连接时 SDK Room 为 nil，桥接仍可安全分发（房间名为空）。
	return client, room
}

// TestBridgeUserDataChatTopicTriggersChatMessageEvent 验证 topic 为 ChatTopic 的
// SDK UserDataPacket 会真实触发 ChatMessageEvent 分发。
func TestBridgeUserDataChatTopicTriggersChatMessageEvent(t *testing.T) {
	client, room := newBridgeTestClient(t)
	got := make(chan ChatMessageEvent, 1)
	client.OnChatMessage(func(_ context.Context, event ChatMessageEvent) error {
		got <- event
		return nil
	})
	cb := client.roomCallback(room, nil)
	cb.OnDataPacket(&lksdk.UserDataPacket{Payload: []byte("hello chat"), Topic: ChatTopic},
		lksdk.DataReceiveParams{SenderIdentity: "alice"})
	select {
	case event := <-got:
		if event.Text != "hello chat" || event.SenderID != "alice" || event.Topic != ChatTopic {
			t.Fatalf("unexpected chat event: %+v", event)
		}
		if string(event.Payload) != "hello chat" {
			t.Fatalf("unexpected payload: %q", event.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("expected ChatMessageEvent from SDK Data callback")
	}
}

// TestBridgeChatMessagePacketTriggersChatMessageEvent 验证 SDK 原生
// *livekit.ChatMessage 数据包会真实触发 ChatMessageEvent 分发。
func TestBridgeChatMessagePacketTriggersChatMessageEvent(t *testing.T) {
	client, room := newBridgeTestClient(t)
	got := make(chan ChatMessageEvent, 1)
	client.OnChatMessage(func(_ context.Context, event ChatMessageEvent) error {
		got <- event
		return nil
	})
	cb := client.roomCallback(room, nil)
	msg := lksdk.ChatMessage(time.Now(), "native chat")
	cb.OnDataPacket(msg, lksdk.DataReceiveParams{SenderIdentity: "bob"})
	select {
	case event := <-got:
		if event.Text != "native chat" || event.SenderID != "bob" || event.Topic != ChatTopic {
			t.Fatalf("unexpected chat event: %+v", event)
		}
		if event.MessageID != msg.GetId() || event.Timestamp != msg.GetTimestamp() {
			t.Fatalf("message identity fields lost: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected ChatMessageEvent from native chat message packet")
	}
}

// TestBridgeNonChatTopicOnlyDispatchesDataEvent 验证非聊天 topic 的 UserDataPacket
// 只分发 DataEvent，不误报为聊天消息。
func TestBridgeNonChatTopicOnlyDispatchesDataEvent(t *testing.T) {
	client, room := newBridgeTestClient(t)
	var chatCalls, dataCalls int
	client.OnChatMessage(func(context.Context, ChatMessageEvent) error { chatCalls++; return nil })
	client.OnData(func(_ context.Context, event DataEvent) error {
		dataCalls++
		if event.Topic != "metrics" || string(event.Payload) != "42" || event.SenderIdentity != "alice" {
			t.Fatalf("unexpected data event: %+v", event)
		}
		return nil
	})
	cb := client.roomCallback(room, nil)
	cb.OnDataPacket(&lksdk.UserDataPacket{Payload: []byte("42"), Topic: "metrics"},
		lksdk.DataReceiveParams{SenderIdentity: "alice"})
	if chatCalls != 0 {
		t.Fatalf("chat hook must not fire for non-chat topic, calls = %d", chatCalls)
	}
	if dataCalls != 1 {
		t.Fatalf("data hook calls = %d, want 1", dataCalls)
	}
}

// TestBridgeRoomAndParticipantCallbacksDispatchTypedHooks 验证 SDK Room/Participant/
// Track 回调桥接到对应 typed Hook。
func TestBridgeRoomAndParticipantCallbacksDispatchTypedHooks(t *testing.T) {
	client, room := newBridgeTestClient(t)
	connected := make(chan ParticipantConnectedEvent, 1)
	disconnected := make(chan ParticipantDisconnectedEvent, 1)
	tracks := make(chan TrackEvent, 4)
	meta := make(chan RoomMetadataEvent, 1)
	connState := make(chan RoomConnectionEvent, 3)
	client.OnParticipantConnected(func(_ context.Context, event ParticipantConnectedEvent) error {
		connected <- event
		return nil
	})
	client.OnParticipantDisconnected(func(_ context.Context, event ParticipantDisconnectedEvent) error {
		disconnected <- event
		return nil
	})
	client.OnTrack(func(_ context.Context, event TrackEvent) error {
		tracks <- event
		return nil
	})
	client.OnRoomMetadata(func(_ context.Context, event RoomMetadataEvent) error {
		meta <- event
		return nil
	})
	client.OnRoomConnection(func(_ context.Context, event RoomConnectionEvent) error {
		connState <- event
		return nil
	})

	cb := client.roomCallback(room, nil)
	cb.OnParticipantConnected(nil)
	cb.OnParticipantDisconnected(nil)
	cb.OnTrackMuted(nil, nil)
	cb.OnTrackPublished(nil, nil)
	cb.OnTrackSubscriptionFailed("TR_SID", nil)
	cb.OnRoomMetadataChanged("meta-v2")
	cb.OnReconnecting()
	cb.OnReconnected()

	select {
	case event := <-connected:
		if event.RoomName != "" || event.Participant != nil {
			t.Fatalf("unexpected connected event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected participant connected hook")
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("expected participant disconnected hook")
	}
	kinds := map[TrackEventKind]bool{}
	for i := 0; i < 3; i++ {
		select {
		case event := <-tracks:
			kinds[event.Kind] = true
		case <-time.After(time.Second):
			t.Fatalf("expected %d track hooks, got %d", 3, i)
		}
	}
	if !kinds[TrackEventMuted] || !kinds[TrackEventPublished] || !kinds[TrackEventSubscriptionFailed] {
		t.Fatalf("unexpected track kinds: %#v", kinds)
	}
	select {
	case event := <-meta:
		if event.Metadata != "meta-v2" {
			t.Fatalf("unexpected metadata event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected room metadata hook")
	}
	statuses := map[RoomConnectionStatus]bool{}
	for i := 0; i < 2; i++ {
		select {
		case event := <-connState:
			statuses[event.Status] = true
		case <-time.After(time.Second):
			t.Fatalf("expected %d connection hooks, got %d", 2, i)
		}
	}
	if !statuses[RoomConnectionReconnecting] || !statuses[RoomConnectionReconnected] {
		t.Fatalf("unexpected connection statuses: %#v", statuses)
	}
}

// TestBridgeDisconnectDispatchesReasonAndCancelsContext 验证断开回调分发
// Disconnected 事件并取消连接级 Hook context。真实断开原因来自 SDK Room，
// 未连接（room 为 nil）时原因为零值；真实原因由集成测试验证。
func TestBridgeDisconnectDispatchesReasonAndCancelsContext(t *testing.T) {
	client, room := newBridgeTestClient(t)
	got := make(chan RoomConnectionEvent, 1)
	client.OnRoomConnection(func(_ context.Context, event RoomConnectionEvent) error {
		got <- event
		return nil
	})
	cb := client.roomCallback(room, nil)
	cb.OnDisconnected()
	select {
	case event := <-got:
		if event.Status != RoomConnectionDisconnected {
			t.Fatalf("unexpected disconnect event: %+v", event)
		}
		if event.ProtocolReason != livekit.DisconnectReason_UNKNOWN_REASON {
			t.Fatalf("unexpected protocol reason for nil room: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected disconnected hook")
	}
	select {
	case <-room.ctx.Done():
	default:
		t.Fatal("expected room context to be canceled after disconnect")
	}
}

// TestBridgeCustomCallbackOverridesBridgedFields 验证调用方自定义回调中非 nil
// 字段按 SDK Merge 语义覆盖内置桥接，未设置字段仍走内置桥接。
func TestBridgeCustomCallbackOverridesBridgedFields(t *testing.T) {
	client, room := newBridgeTestClient(t)
	var bridgedCalls, customCalls int
	client.OnParticipantConnected(func(context.Context, ParticipantConnectedEvent) error {
		bridgedCalls++
		return nil
	})
	custom := &lksdk.RoomCallback{
		OnParticipantConnected: func(*lksdk.RemoteParticipant) {
			customCalls++
		},
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: func(lksdk.DataPacket, lksdk.DataReceiveParams) {},
		},
	}
	cb := client.roomCallback(room, custom)
	// 覆盖字段：只调用自定义回调。
	cb.OnParticipantConnected(nil)
	if customCalls != 1 || bridgedCalls != 0 {
		t.Fatalf("custom calls = %d, bridged calls = %d", customCalls, bridgedCalls)
	}
	// 未覆盖字段：仍走内置桥接。
	cb.OnRoomMetadataChanged("meta")
	if bridgedCalls != 0 {
		t.Fatalf("participant hook must not fire for metadata change: %d", bridgedCalls)
	}
	// 自定义回调覆盖 OnDataPacket 后，聊天消息不再自动分发。
	var chatCalls int
	client.OnChatMessage(func(context.Context, ChatMessageEvent) error { chatCalls++; return nil })
	cb.OnDataPacket(&lksdk.UserDataPacket{Payload: []byte("x"), Topic: ChatTopic},
		lksdk.DataReceiveParams{SenderIdentity: "alice"})
	if chatCalls != 0 {
		t.Fatalf("chat hook must not fire when OnDataPacket is overridden, calls = %d", chatCalls)
	}
}

// TestBridgeDispatchConcurrentSafety 验证桥接分发与注册/注销并发执行无数据竞争。
func TestBridgeDispatchConcurrentSafety(t *testing.T) {
	client, room := newBridgeTestClient(t)
	var wg sync.WaitGroup
	cb := client.roomCallback(room, nil)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个 goroutine 持有自己的订阅句柄，避免测试自身的数据竞争。
			sub := client.OnData(func(context.Context, DataEvent) error { return nil })
			defer sub.Unsubscribe()
			for j := 0; j < 30; j++ {
				cb.OnDataPacket(&lksdk.UserDataPacket{Payload: []byte("x"), Topic: "t"},
					lksdk.DataReceiveParams{SenderIdentity: "alice"})
				sub.Unsubscribe()
				sub = client.OnData(func(context.Context, DataEvent) error { return nil })
			}
		}()
	}
	wg.Wait()
}

// TestBridgeRPCRequestDispatch 验证 RegisterRPC 包装后的 handler 会分发
// RPCRequestEvent Hook（在真实连接中由 SDK OnRpcRequest 触发）。
// CallerIdentity/RequestID 来自 SDK 注入 context 的 RpcInvocationMetadata，
// 由真实集成测试验证。
func TestBridgeRPCRequestDispatch(t *testing.T) {
	client, room := newBridgeTestClient(t)
	got := make(chan RPCRequestEvent, 1)
	client.OnRPCRequest(func(_ context.Context, event RPCRequestEvent) error {
		got <- event
		return nil
	})
	room.room = lksdk.NewRoom(&lksdk.RoomCallback{})
	if err := room.RegisterRPC("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}); err != nil {
		t.Fatal(err)
	}
	room.dispatchRPCRequest(context.Background(), "echo", []byte("ping"))
	select {
	case event := <-got:
		if event.Method != "echo" || event.Payload != "ping" {
			t.Fatalf("unexpected rpc event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected RPC request hook")
	}
}
