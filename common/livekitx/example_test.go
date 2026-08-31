package livekitx_test

// 本文件编译验证 README 中的使用方式；Example 函数没有 Output 注释，
// 只参与编译不执行，避免示例代码在测试中产生真实网络请求。

import (
	"context"
	"net/http"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"zero-service/common/livekitx"
)

func Example_initialize() {
	httpClient := &http.Client{}
	client, err := livekitx.New(
		livekitx.WithURL("http://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
		livekitx.WithHTTPClient(httpClient),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rooms, err := client.API().Room().ListRooms(ctx, nil)
	_, _ = rooms, err
}

// Example_join 演示 Connect 内置回调桥与自定义回调叠加。
func Example_join() {
	client, err := livekitx.New(
		livekitx.WithURL("http://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx := context.Background()
	token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
		APIKey: "devkey", APISecret: "secret",
		Room: "demo", Identity: "user-a", CanPublish: true, CanSubscribe: true, CanPublishData: true,
	})
	if err != nil {
		panic(err)
	}
	// 传 nil 完整保留内置回调桥，业务通过 client.OnXXX 注册 Hook。
	room, err := client.Connect(ctx, token, nil)
	if err != nil {
		panic(err)
	}
	defer room.Close()
	// 叠加自定义回调：只覆盖 OnParticipantConnected，其余字段仍走内置桥接。
	_, _ = client.Connect(ctx, token, &lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {},
	})
}

// Example_hooks 演示 Hook 注册与注销。
func Example_hooks() {
	client, err := livekitx.New(
		livekitx.WithURL("http://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	chatSub := client.OnChatMessage(func(_ context.Context, event livekitx.ChatMessageEvent) error {
		// 落库、未读数、敏感词审核由业务实现。
		_, _, _ = event.RoomName, event.SenderID, event.Text
		return nil
	})
	webhookSub := client.OnWebhook(func(_ context.Context, event *livekitx.WebhookEvent) error {
		// 持久化 event.GetId() 做幂等，处理对账与未知事件。
		return nil
	})
	client.OnRoomConnection(func(_ context.Context, event livekitx.RoomConnectionEvent) error {
		// 会议状态同步：connected/reconnecting/reconnected/disconnected。
		return nil
	})
	_ = chatSub
	_ = webhookSub
}

// Example_webhook 演示 ReceiveWebhook 校验与分发。
func Example_webhook() {
	client, err := livekitx.New(
		livekitx.WithURL("http://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	// 收到 HTTP 请求后；验签失败返回错误且不调用任何 handler。
	var httpRequest *http.Request
	err = client.ReceiveWebhook(context.Background(), httpRequest, "webhook-signing-key", nil)
	_ = err
}

// Example_chatAndRPC 演示聊天发送与 RPC 注册/调用。
func Example_chatAndRPC() {
	client, err := livekitx.New(
		livekitx.WithURL("http://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	var room *livekitx.RealtimeRoom
	// 发送 SDK 原生聊天消息；接收方通过 OnChatMessage 收到。
	_ = room.PublishChatMessage("hello")
	// 注册 RPC handler；请求到达时先触发 OnRPCRequest Hook。
	_ = room.RegisterRPC("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	})
	// 调用远端 RPC；ResponseTimeout 控制时限。
	timeout := 10 * time.Second
	_, _ = room.PerformRPC(lksdk.PerformRpcParams{
		DestinationIdentity: "user-b", Method: "echo", Payload: "ping", ResponseTimeout: &timeout,
	})
	_ = client
}
