package livekitx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/proto"
)

// newAPIMockServer 构造校验 Twirp 管理请求的 mock 服务端：
// 断言 HTTP 方法、路径、Authorization 与请求体，并返回对应 protobuf 响应。
func newAPIMockServer(t *testing.T, secret string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			t.Error("missing Authorization header")
		}
		verifier, err := auth.ParseAPIToken(raw)
		if err != nil {
			t.Errorf("invalid auth token: %v", err)
		} else if _, grants, err := verifier.Verify(secret); err != nil {
			t.Errorf("auth token verification failed: %v", err)
		} else if grants.Video == nil {
			t.Error("auth token missing video grant")
		} else {
			verifyGrants(t, r.URL.Path, grants)
		}
		switch r.URL.Path {
		case "/twirp/livekit.RoomService/ListRooms":
			var req livekit.ListRoomsRequest
			readProto(t, r, &req)
			writeProto(t, w, &livekit.ListRoomsResponse{})
		case "/twirp/livekit.RoomService/ListParticipants":
			var req livekit.ListParticipantsRequest
			readProto(t, r, &req)
			if req.Room != "room-a" {
				t.Errorf("request room = %q, want room-a", req.Room)
			}
			writeProto(t, w, &livekit.ListParticipantsResponse{})
		case "/twirp/livekit.Egress/ListEgress":
			var req livekit.ListEgressRequest
			readProto(t, r, &req)
			writeProto(t, w, &livekit.ListEgressResponse{})
		case "/twirp/livekit.Ingress/ListIngress":
			var req livekit.ListIngressRequest
			readProto(t, r, &req)
			writeProto(t, w, &livekit.ListIngressResponse{})
		case "/twirp/livekit.SIP/ListSIPTrunk":
			var req livekit.ListSIPTrunkRequest
			readProto(t, r, &req)
			writeProto(t, w, &livekit.ListSIPTrunkResponse{})
		case "/twirp/livekit.AgentDispatchService/ListDispatch":
			var req livekit.ListAgentDispatchRequest
			readProto(t, r, &req)
			writeProto(t, w, &livekit.ListAgentDispatchResponse{})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// verifyGrants 按 Twirp 路径断言请求 token 携带的授权权限。
func verifyGrants(t *testing.T, path string, grants *auth.ClaimGrants) {
	t.Helper()
	switch path {
	case "/twirp/livekit.RoomService/ListRooms":
		if !grants.Video.RoomList {
			t.Error("ListRooms token missing roomList grant")
		}
	case "/twirp/livekit.RoomService/ListParticipants":
		if !grants.Video.RoomAdmin || grants.Video.Room != "room-a" {
			t.Errorf("ListParticipants token grants = %+v, want roomAdmin scoped to room-a", grants.Video)
		}
	case "/twirp/livekit.Egress/ListEgress":
		if !grants.Video.RoomRecord {
			t.Error("ListEgress token missing roomRecord grant")
		}
	case "/twirp/livekit.Ingress/ListIngress":
		if !grants.Video.IngressAdmin {
			t.Error("ListIngress token missing ingressAdmin grant")
		}
	case "/twirp/livekit.SIP/ListSIPTrunk":
		if grants.SIP == nil || !grants.SIP.Admin {
			t.Errorf("ListSIPTrunk token sip grants = %+v, want admin", grants.SIP)
		}
	case "/twirp/livekit.AgentDispatchService/ListDispatch":
		if !grants.Video.RoomAdmin {
			t.Error("ListDispatch token missing roomAdmin grant")
		}
	}
}

// readProto 读取请求体并按 protobuf 解码。
func readProto(t *testing.T, r *http.Request, msg proto.Message) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.Unmarshal(body, msg); err != nil {
		t.Fatalf("decode %T: %v", msg, err)
	}
}

// writeProto 以 application/protobuf 返回 protobuf 响应。
func writeProto(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	body, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/protobuf")
	_, _ = w.Write(body)
}

// writeTwirpError 以 twirp JSON 错误格式返回错误响应。
func writeTwirpError(t *testing.T, w http.ResponseWriter, err twirp.Error) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(twirp.ServerHTTPStatusFromErrorCode(err.Code()))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code": string(err.Code()),
		"msg":  err.Msg(),
		"meta": map[string]string{},
	})
}

// TestAPIManagementClientsAuthenticateAndRoute 验证五个管理 service 的
// 请求方法、路径、Authorization 签名和请求体都真实到达 Twirp 端点。
func TestAPIManagementClientsAuthenticateAndRoute(t *testing.T) {
	server := newAPIMockServer(t, "secret")
	client, err := New(WithURL(server.URL), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()

	if _, err = client.API().Room().ListRooms(ctx, &livekit.ListRoomsRequest{Names: []string{"demo"}}); err != nil {
		t.Fatalf("ListRooms: %v", err)
	}
	if _, err = client.API().Room().ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: "room-a"}); err != nil {
		t.Fatalf("ListParticipants: %v", err)
	}
	if _, err = client.API().Egress().ListEgress(ctx, &livekit.ListEgressRequest{}); err != nil {
		t.Fatalf("ListEgress: %v", err)
	}
	if _, err = client.API().Ingress().ListIngress(ctx, &livekit.ListIngressRequest{}); err != nil {
		t.Fatalf("ListIngress: %v", err)
	}
	if _, err = client.API().SIP().ListSIPTrunk(ctx, &livekit.ListSIPTrunkRequest{}); err != nil {
		t.Fatalf("ListSIPTrunk: %v", err)
	}
	if _, err = client.API().AgentDispatch().ListDispatch(ctx, &livekit.ListAgentDispatchRequest{}); err != nil {
		t.Fatalf("ListDispatch: %v", err)
	}
}

// TestAPIManagementRequestUsesInjectedHTTPClient 验证注入的 HTTP client
// 真正执行管理请求（在 TestNewUsesInjectedHTTPService 基础上补充
// 标准 *http.Client 路径的真实往返）。
func TestAPIManagementRequestUsesInjectedHTTPClient(t *testing.T) {
	server := newAPIMockServer(t, "secret")
	client, err := New(WithURL(server.URL), WithAPIKey("devkey", "secret"), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err = client.API().Room().ListRooms(context.Background(), &livekit.ListRoomsRequest{}); err != nil {
		t.Fatalf("ListRooms with injected client: %v", err)
	}
}

// TestAPITwirpErrorPreservesCodeAndMessage 验证 Twirp 错误原样保留
// code/message，可通过 errors.As 判断 twirp.Error。
func TestAPITwirpErrorPreservesCodeAndMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTwirpError(t, w, twirp.NewError(twirp.PermissionDenied, "no permission"))
	}))
	defer server.Close()
	client, err := New(WithURL(server.URL), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, err = client.API().Room().ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err == nil {
		t.Fatal("expected twirp error")
	}
	var twirpErr twirp.Error
	if !errors.As(err, &twirpErr) {
		t.Fatalf("expected twirp.Error, got %T: %v", err, err)
	}
	if twirpErr.Code() != twirp.PermissionDenied || twirpErr.Msg() != "no permission" {
		t.Fatalf("unexpected twirp error: code=%s msg=%q", twirpErr.Code(), twirpErr.Msg())
	}
}

// TestAPIContextCancellationPropagates 验证已取消的调用方 context 会中止
// 管理请求并返回 context.Canceled，而不是发起网络请求。
func TestAPIContextCancellationPropagates(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		writeProto(t, w, &livekit.ListRoomsResponse{})
	}))
	defer server.Close()
	client, err := New(WithURL(server.URL), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.API().Room().ListRooms(ctx, &livekit.ListRoomsRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if called {
		t.Fatal("request must not reach server with canceled context")
	}
}
