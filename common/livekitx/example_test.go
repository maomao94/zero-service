package livekitx_test

// 本文件编译验证 README 与 docs/live/callbacks-guide.md 中的使用方式；
// Example 函数没有 Output 注释，只参与编译不执行，避免示例代码在测试中
// 产生真实网络请求。

import (
	"context"
	"net/http"
	"time"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"zero-service/common/livekitx"
)

func Example_initialize() {
	httpClient := &http.Client{}
	client, err := livekitx.New(
		livekitx.WithURL("https://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
		livekitx.WithHTTPClient(httpClient),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rooms, err := client.Room().ListRooms(ctx, &livekit.ListRoomsRequest{})
	_, _ = rooms, err
}

// Example_join 演示使用 SDK 原生方式加入房间：
// 业务直接用 lksdk.NewRoom + JoinWithContext，livekitx 只提供 Client 管理配置。
func Example_join() {
	client, err := livekitx.New(
		livekitx.WithURL("https://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx := context.Background()
	room := lksdk.NewRoom(&lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
			// 参会名单更新、入会通知。
		},
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
				// 聊天/Data/RPC 数据都在这里识别。
			},
		},
		OnDisconnected: func() { /* 本次连接的清理逻辑 */ },
	})
	if err := room.JoinWithContext(ctx, "https://127.0.0.1:7880", lksdk.ConnectInfo{
		APIKey:              "devkey",
		APISecret:           "secret",
		RoomName:            "demo",
		ParticipantIdentity: "user-a",
	}, lksdk.WithAutoSubscribe(true)); err != nil {
		panic(err)
	}
	defer room.Disconnect()
}

// Example_roomAPI 演示 Room API：通过 client.Room() 直接使用 SDK 原生方法。
func Example_roomAPI() {
	client, err := livekitx.New(
		livekitx.WithURL("https://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx := context.Background()
	roomSvc := client.Room()
	// 创建房间
	info, err := roomSvc.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "demo"})
	_, _ = info, err
	// 踢人
	_, err = roomSvc.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: "demo", Identity: "user-b"})
	_ = err
	// 发送 Data（广播）
	topic := "chat-media"
	_, err = roomSvc.SendData(ctx, &livekit.SendDataRequest{
		Room:  "demo",
		Topic: &topic,
		Data:  []byte("voice bytes"),
		Kind:  livekit.DataPacket_RELIABLE,
	})
	_ = err
	// 发送 Data（定向）
	_, err = roomSvc.SendData(ctx, &livekit.SendDataRequest{
		Room:                  "demo",
		Topic:                 &topic,
		Data:                  []byte("image bytes"),
		Kind:                  livekit.DataPacket_RELIABLE,
		DestinationIdentities: []string{"user-b"},
	})
	_ = err
	// 结束会议/删除房间
	_, err = roomSvc.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: "demo"})
	_ = err
}

// Example_joinToken 演示使用 livekitx.NewJoinToken 签发入会 token：
// 业务在用户鉴权后发放给客户端 App，由客户端直接连接 LiveKit。
func Example_joinToken() {
	token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
		APIKey:         "devkey",
		APISecret:      "secret",
		Room:           "demo",
		Identity:       "user-a",
		ValidFor:       2 * time.Hour,
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
	})
	if err != nil {
		panic(err)
	}
	// token 只发放给已鉴权客户端；不写入日志。
	_ = token
}

// Example_mute 演示经管理 API 静音/取消静音参与者全部音频轨道。
func Example_mute() {
	client, err := livekitx.New(
		livekitx.WithURL("https://127.0.0.1:7880"),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx := context.Background()
	// 列出参与者，找到目标的音频轨道，逐个静音
	participants, _ := client.Room().ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: "demo"})
	for _, p := range participants.GetParticipants() {
		if p.Identity != "user-b" {
			continue
		}
		for _, track := range p.Tracks {
			if track.Type != livekit.TrackType_AUDIO {
				continue
			}
			_, _ = client.Room().MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
				Room:     "demo",
				Identity: "user-b",
				TrackSid: track.Sid,
				Muted:    true,
			})
		}
	}
}

// Example_webhook 演示 Webhook 验签与业务侧处理。
func Example_webhook() {
	var httpRequest *http.Request
	event, err := webhook.ReceiveWebhookEvent(httpRequest, livekitx.NewWebhookKeyProvider("webhook-signing-key"))
	if err != nil {
		return
	}
	_, _ = event.GetEvent(), event.GetId()
}

// Example_chatAndRPC 演示 SDK 原生聊天发送与 RPC 注册/调用。
func Example_chatAndRPC() {
	var room *lksdk.Room // 来自 lksdk.NewRoom + JoinWithContext
	// 发送 SDK 原生聊天消息
	_ = room.LocalParticipant.PublishDataPacket(lksdk.ChatMessage(time.Now(), "hello"))
	// 注册 RPC handler
	_ = room.RegisterRpcCtxMethod("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	})
	// 调用远端 RPC
	timeout := 10 * time.Second
	_, _ = room.LocalParticipant.PerformRpc(lksdk.PerformRpcParams{
		DestinationIdentity: "user-b", Method: "echo", Payload: "ping", ResponseTimeout: &timeout,
	})
}
