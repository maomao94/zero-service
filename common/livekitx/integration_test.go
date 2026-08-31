package livekitx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

func TestLiveKitDevServerRoomLifecycle(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := New(WithURL(envOr("LIVEKIT_URL", "http://127.0.0.1:7880")), WithAPIKey(envOr("LIVEKIT_API_KEY", "devkey"), envOr("LIVEKIT_API_SECRET", "secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	name := uniqueRoomName("lifecycle")
	if _, err = client.API().Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}
	token, err := NewJoinToken(JoinTokenOptions{APIKey: envOr("LIVEKIT_API_KEY", "devkey"), APISecret: envOr("LIVEKIT_API_SECRET", "secret"), Room: name, Identity: "integration-user", CanPublish: true, CanSubscribe: true})
	if err != nil {
		t.Fatal(err)
	}
	realtime, err := client.Connect(ctx, token, &lksdk.RoomCallback{})
	if err != nil {
		t.Fatal(err)
	}
	defer realtime.Close()
	participants, err := client.API().Room().ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: name})
	if err != nil {
		t.Fatal(err)
	}
	if len(participants.Participants) != 1 || participants.Participants[0].Identity != "integration-user" {
		t.Fatalf("participants = %#v", participants.Participants)
	}
	if _, err = client.API().Room().ListRooms(ctx, &livekit.ListRoomsRequest{Names: []string{name}}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.API().Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

// TestLiveKitDevServerRealtimeBridgeAndChat 验证真实服务端上的桥接链路：
// 参与者入会 Hook、Data 回调触发 ChatMessageEvent、断开事件带真实原因、
// 以及 RPC 请求 Hook 与真实调用往返。
func TestLiveKitDevServerRealtimeBridgeAndChat(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	client, err := New(WithURL(envOr("LIVEKIT_URL", "http://127.0.0.1:7880")), WithAPIKey(envOr("LIVEKIT_API_KEY", "devkey"), envOr("LIVEKIT_API_SECRET", "secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	name := uniqueRoomName("hooks")
	if _, err = client.API().Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}

	// 注册桥接 Hook：alice 侧观察 bob 入会；bob 侧接收聊天与 RPC 请求。
	bobJoined := make(chan ParticipantConnectedEvent, 1)
	client.OnParticipantConnected(func(_ context.Context, event ParticipantConnectedEvent) error {
		if event.Participant != nil && event.Participant.Identity() == "bob" {
			bobJoined <- event
		}
		return nil
	})
	chat := make(chan ChatMessageEvent, 1)
	client.OnChatMessage(func(_ context.Context, event ChatMessageEvent) error {
		chat <- event
		return nil
	})
	rpcReq := make(chan RPCRequestEvent, 1)
	client.OnRPCRequest(func(_ context.Context, event RPCRequestEvent) error {
		rpcReq <- event
		return nil
	})
	disconnected := make(chan RoomConnectionEvent, 1)
	client.OnRoomConnection(func(_ context.Context, event RoomConnectionEvent) error {
		if event.Status == RoomConnectionDisconnected {
			disconnected <- event
		}
		return nil
	})

	key := envOr("LIVEKIT_API_KEY", "devkey")
	secret := envOr("LIVEKIT_API_SECRET", "secret")
	roomA, err := client.Connect(ctx, joinToken(t, key, secret, name, "alice"), nil)
	if err != nil {
		t.Fatal(err)
	}
	roomB, err := client.Connect(ctx, joinToken(t, key, secret, name, "bob"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer roomA.Close()
	defer roomB.Close()

	// alice 的内置桥接应真实收到 bob 入会事件。
	select {
	case event := <-bobJoined:
		if event.Participant.Identity() != "bob" {
			t.Fatalf("unexpected participant: %s", event.Participant.Identity())
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for participant joined hook")
	}

	// bob 注册 RPC，alice 真实调用；OnRPCRequest Hook 应带 SDK 元数据。
	if err := roomB.RegisterRPC("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}); err != nil {
		t.Fatal(err)
	}
	timeout := 10 * time.Second
	response, err := roomA.PerformRPC(lksdk.PerformRpcParams{
		DestinationIdentity: "bob", Method: "echo", Payload: "ping", ResponseTimeout: &timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || *response != "ping" {
		t.Fatalf("rpc response = %v", response)
	}
	select {
	case event := <-rpcReq:
		if event.Method != "echo" || event.Payload != "ping" || event.CallerIdentity != "alice" {
			t.Fatalf("unexpected rpc hook event: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for rpc request hook")
	}

	// alice 发送 SDK 原生聊天消息，bob 的内置桥接应真实触发 ChatMessageEvent。
	if err := roomA.PublishChatMessage("hello from alice"); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-chat:
		if event.Text != "hello from alice" || event.SenderID != "alice" {
			t.Fatalf("unexpected chat event: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for chat message hook")
	}

	// 主动断开后应收到带真实原因的 Disconnected 事件。
	roomB.Close()
	select {
	case event := <-disconnected:
		if event.ProtocolReason != livekit.DisconnectReason_CLIENT_INITIATED {
			t.Fatalf("unexpected disconnect reason: %+v", event)
		}
		if event.Reason != lksdk.LeaveRequested {
			t.Fatalf("unexpected collapsed reason: %q", event.Reason)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for disconnected hook")
	}

	if _, err = client.API().Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

// TestLiveKitDevServerRealtimeChatTopicData 验证 UserData + ChatTopic 约定在
// 真实服务端同样触发 ChatMessageEvent。
func TestLiveKitDevServerRealtimeChatTopicData(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	client, err := New(WithURL(envOr("LIVEKIT_URL", "http://127.0.0.1:7880")), WithAPIKey(envOr("LIVEKIT_API_KEY", "devkey"), envOr("LIVEKIT_API_SECRET", "secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	name := uniqueRoomName("chattopic")
	if _, err = client.API().Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}
	chat := make(chan ChatMessageEvent, 1)
	client.OnChatMessage(func(_ context.Context, event ChatMessageEvent) error {
		chat <- event
		return nil
	})
	key := envOr("LIVEKIT_API_KEY", "devkey")
	secret := envOr("LIVEKIT_API_SECRET", "secret")
	roomA, err := client.Connect(ctx, joinToken(t, key, secret, name, "alice"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer roomA.Close()
	roomB, err := client.Connect(ctx, joinToken(t, key, secret, name, "bob"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer roomB.Close()
	if err := roomA.PublishDataPacket(&lksdk.UserDataPacket{Payload: []byte("legacy chat"), Topic: ChatTopic}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-chat:
		if event.Text != "legacy chat" || event.SenderID != "alice" || event.Topic != ChatTopic {
			t.Fatalf("unexpected chat event: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for chat topic data hook")
	}
	if _, err = client.API().Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

// envOr 返回环境变量值，未设置时使用本地 dev server 默认值。
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// uniqueRoomName 生成带前缀和纳秒时间戳的唯一房间名，避免集成测试间冲突。
func uniqueRoomName(prefix string) string {
	return "livekitx-" + prefix + "-" + time.Now().Format("150405.000000000")
}

// joinToken 生成指定房间和身份、可发布/订阅/发数据的参与者 token。
func joinToken(t *testing.T, key, secret, room, identity string) string {
	t.Helper()
	token, err := NewJoinToken(JoinTokenOptions{
		APIKey: key, APISecret: secret, Room: room, Identity: identity,
		CanPublish: true, CanSubscribe: true, CanPublishData: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}
