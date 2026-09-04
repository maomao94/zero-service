package livekitx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const chatUserDataTopic = "test-chat"

// joinRoomHelper 用 SDK 原生方式加入房间，替代已删除的 client.JoinRoom。
func joinRoomHelper(t *testing.T, client *Client, roomName, identity string, cb *lksdk.RoomCallback, connectOpts ...lksdk.ConnectOption) *lksdk.Room {
	t.Helper()
	room := lksdk.NewRoom(cb)
	if err := room.JoinWithContext(t.Context(), client.config.URL, lksdk.ConnectInfo{
		APIKey:              client.config.APIKey,
		APISecret:           client.config.APISecret,
		RoomName:            roomName,
		ParticipantIdentity: identity,
	}, connectOpts...); err != nil {
		t.Fatal(err)
	}
	return room
}

func TestLiveKitDevServerRoomLifecycle(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := New(WithURL(envOr("LIVEKIT_URL", "https://127.0.0.1:7880")), WithAPIKey(envOr("LIVEKIT_API_KEY", "devkey"), envOr("LIVEKIT_API_SECRET", "secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	name := uniqueRoomName("lifecycle")
	if _, err = client.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}
	room := joinRoomHelper(t, client, name, "integration-user", nil)
	defer room.Disconnect()
	participants, err := client.Room().ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: name})
	if err != nil {
		t.Fatal(err)
	}
	if len(participants.Participants) != 1 || participants.Participants[0].Identity != "integration-user" {
		t.Fatalf("participants = %#v", participants.Participants)
	}
	if _, err = client.Room().ListRooms(ctx, &livekit.ListRoomsRequest{Names: []string{name}}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

func TestLiveKitDevServerNativeCallbacksAndChat(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	url := envOr("LIVEKIT_URL", "https://127.0.0.1:7880")
	key := envOr("LIVEKIT_API_KEY", "devkey")
	secret := envOr("LIVEKIT_API_SECRET", "secret")

	bobJoined := make(chan string, 1)
	bobLeft := make(chan string, 1)
	chat := make(chan *livekit.ChatMessage, 1)
	invite := make(chan *lksdk.UserDataPacket, 1)
	media := make(chan *lksdk.UserDataPacket, 1)
	disconnected := make(chan livekit.DisconnectReason, 1)
	var bobRoom *lksdk.Room

	clientA, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientA.Close()
	clientB, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientB.Close()
	name := uniqueRoomName("native")
	if _, err = clientA.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}

	roomA := joinRoomHelper(t, clientA, name, "alice", &lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
			if rp.Identity() == "bob" {
				bobJoined <- rp.Identity()
			}
		},
		OnParticipantDisconnected: func(rp *lksdk.RemoteParticipant) {
			if rp.Identity() == "bob" {
				bobLeft <- rp.Identity()
			}
		},
	})
	defer roomA.Disconnect()

	roomB := joinRoomHelper(t, clientB, name, "bob", &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
				if msg, ok := data.(*livekit.ChatMessage); ok && msg.GetMessage() == "hello from alice" {
					chat <- msg
				}
				if msg, ok := data.(*lksdk.UserDataPacket); ok && msg.Topic == "invite" {
					invite <- msg
				}
				if msg, ok := data.(*lksdk.UserDataPacket); ok && msg.Topic == "media" {
					media <- msg
				}
			},
		},
		OnDisconnected: func() {
			disconnected <- bobRoom.DisconnectReason()
		},
	})
	defer roomB.Disconnect()
	bobRoom = roomB

	select {
	case identity := <-bobJoined:
		if identity != "bob" {
			t.Fatalf("unexpected participant: %s", identity)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for OnParticipantConnected")
	}

	if err := bobRoom.RegisterRpcCtxMethod("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}); err != nil {
		t.Fatal(err)
	}
	timeout := 10 * time.Second
	response, err := roomA.LocalParticipant.PerformRpc(lksdk.PerformRpcParams{
		DestinationIdentity: "bob", Method: "echo", Payload: "ping", ResponseTimeout: &timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || *response != "ping" {
		t.Fatalf("rpc response = %v", response)
	}

	if err := roomA.LocalParticipant.PublishDataPacket(lksdk.ChatMessage(time.Now(), "hello from alice")); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-chat:
		if msg.GetId() == "" || msg.GetTimestamp() == 0 {
			t.Fatalf("chat message identity fields missing: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for native chat message")
	}

	topic := "invite"
	if _, err := clientA.Room().SendData(ctx, &livekit.SendDataRequest{
		Room:                  name,
		Topic:                 &topic,
		Data:                  []byte("join now"),
		Kind:                  livekit.DataPacket_RELIABLE,
		DestinationIdentities: []string{"bob"},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-invite:
		if string(msg.Payload) != "join now" || msg.Topic != "invite" {
			t.Fatalf("unexpected invite data: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for invite data")
	}

	mediaTopic := "media"
	if _, err := clientA.Room().SendData(ctx, &livekit.SendDataRequest{
		Room:  name,
		Topic: &mediaTopic,
		Data:  []byte("voice bytes"),
		Kind:  livekit.DataPacket_RELIABLE,
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-media:
		if string(msg.Payload) != "voice bytes" || msg.Topic != "media" {
			t.Fatalf("unexpected media data: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for media data")
	}

	if _, err := clientA.Room().RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: name, Identity: "bob"}); err != nil {
		t.Fatal(err)
	}
	select {
	case reason := <-disconnected:
		if reason != livekit.DisconnectReason_PARTICIPANT_REMOVED {
			t.Fatalf("unexpected disconnect reason: %v", reason)
		}
		if collapsed := lksdk.GetDisconnectionReason(reason); collapsed != lksdk.ParticipantRemoved {
			t.Fatalf("unexpected collapsed reason: %q", collapsed)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for OnDisconnected")
	}
	select {
	case identity := <-bobLeft:
		if identity != "bob" {
			t.Fatalf("unexpected left participant: %s", identity)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for OnParticipantDisconnected")
	}

	if _, err = clientA.Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

func TestLiveKitDevServerRealtimeUserDataChat(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	url := envOr("LIVEKIT_URL", "https://127.0.0.1:7880")
	key := envOr("LIVEKIT_API_KEY", "devkey")
	secret := envOr("LIVEKIT_API_SECRET", "secret")
	chat := make(chan *lksdk.UserDataPacket, 1)

	clientA, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientA.Close()
	clientB, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientB.Close()
	name := uniqueRoomName("chatdata")
	if _, err = clientA.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}
	roomA := joinRoomHelper(t, clientA, name, "alice", nil)
	defer roomA.Disconnect()
	roomB := joinRoomHelper(t, clientB, name, "bob", &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
				if msg, ok := data.(*lksdk.UserDataPacket); ok && msg.Topic == chatUserDataTopic {
					chat <- msg
				}
			},
		},
	})
	defer roomB.Disconnect()

	if err := roomA.LocalParticipant.PublishDataPacket(&lksdk.UserDataPacket{Payload: []byte("legacy chat"), Topic: chatUserDataTopic}); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-chat:
		if string(msg.Payload) != "legacy chat" || msg.Topic != chatUserDataTopic {
			t.Fatalf("unexpected chat topic data: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for chat topic data")
	}
	if _, err = clientA.Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

func TestLiveKitDevServerPerJoinCallback(t *testing.T) {
	if os.Getenv("LIVEKITX_INTEGRATION") != "1" {
		t.Skip("set LIVEKITX_INTEGRATION=1 to run against livekit-server --dev")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	url := envOr("LIVEKIT_URL", "https://127.0.0.1:7880")
	key := envOr("LIVEKIT_API_KEY", "devkey")
	secret := envOr("LIVEKIT_API_SECRET", "secret")

	joined := make(chan string, 1)
	clientA, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientA.Close()
	clientB, err := New(WithURL(url), WithAPIKey(key, secret))
	if err != nil {
		t.Fatal(err)
	}
	defer clientB.Close()
	name := uniqueRoomName("perjoincb")
	if _, err = clientA.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}
	roomA := joinRoomHelper(t, clientA, name, "alice", &lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
			if rp.Identity() == "bob" {
				joined <- rp.Identity()
			}
		},
	})
	defer roomA.Disconnect()
	roomB := joinRoomHelper(t, clientB, name, "bob", nil)
	defer roomB.Disconnect()

	select {
	case identity := <-joined:
		if identity != "bob" {
			t.Fatalf("unexpected participant: %s", identity)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for per-join OnParticipantConnected")
	}
	if _, err = clientA.Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name}); err != nil {
		t.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func uniqueRoomName(prefix string) string {
	return "livekitx-" + prefix + "-" + time.Now().Format("150405.000000000")
}
