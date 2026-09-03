package livekitx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
)

// TestServerRpcToSelf 验证用户报告的场景：客户端注册 echo 后，
// 服务端 RoomService.PerformRpc 定向调用该客户端自身（自发自收）。
func TestServerRpcToSelf(t *testing.T) {
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
	name := uniqueRoomName("rpc-self")
	if _, err = client.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: name}); err != nil {
		t.Fatal(err)
	}

	// 用户身份加入并注册 echo（对应 web 端 "注册 Echo (SDK)" 按钮）
	room := joinRoomHelper(t, client, name, "user-self", nil)
	defer room.Disconnect()
	if err := room.RegisterRpcCtxMethod("echo", func(ctx context.Context, data []byte) ([]byte, error) {
		return append([]byte("pong:"), data...), nil
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	// 场景1：服务端 RPC 调用已注册的自己 → 应成功
	resp, err := client.Room().PerformRpc(ctx, &livekit.PerformRpcRequest{
		Room:                name,
		DestinationIdentity: "user-self",
		Method:              "echo",
		Payload:             "hello data",
		ResponseTimeoutMs:   5000,
	})
	if err != nil {
		t.Fatalf("self RPC should succeed, got error: %v", err)
	}
	if resp.GetPayload() != "pong:hello data" {
		t.Fatalf("unexpected response: %q", resp.GetPayload())
	}
	t.Logf("场景1 通过: 服务端→已注册自身, response=%q", resp.GetPayload())

	// 场景2：注销后再次调用 → 应返回 Method not supported
	room.UnregisterRpcMethod("echo")
	time.Sleep(500 * time.Millisecond)
	if _, err = client.Room().PerformRpc(ctx, &livekit.PerformRpcRequest{
		Room:                name,
		DestinationIdentity: "user-self",
		Method:              "echo",
		Payload:             "hello data",
		ResponseTimeoutMs:   5000,
	}); err == nil {
		t.Fatal("unregistered method should fail")
	} else {
		t.Logf("场景2 符合预期: 注销后报错 = %v", err)
	}

	// 场景3：调用不存在的其他参与者 → 应报错
	if _, err = client.Room().PerformRpc(ctx, &livekit.PerformRpcRequest{
		Room:                name,
		DestinationIdentity: "not-exist-user",
		Method:              "echo",
		Payload:             "hello data",
		ResponseTimeoutMs:   5000,
	}); err == nil {
		t.Fatal("rpc to non-existent participant should fail")
	} else {
		t.Logf("场景3 符合预期: 不存在参与者报错 = %v", err)
	}
}
