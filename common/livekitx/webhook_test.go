package livekitx

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
)

// signedWebhookRequest 构造携带合法 Authorization 签名的 Webhook 请求；
// 签名方式与 livekit-server 一致：API key（devkey）+ body SHA256 放入
// API token 的 Sha256 claim，secret 即 webhook signing key。
func signedWebhookRequest(t *testing.T, body, key string) *http.Request {
	t.Helper()
	hash := sha256.Sum256([]byte(body))
	signature := base64.StdEncoding.EncodeToString(hash[:])
	token, err := auth.NewAccessToken("devkey", key).SetSha256(signature).ToJWT()
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://localhost/webhook", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", token)
	return request
}

// newWebhookTestClient 构造 Webhook 测试 client。
func newWebhookTestClient(t *testing.T) *Client {
	t.Helper()
	client, err := New(WithURL("http://127.0.0.1:7880"), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestReceiveWebhookDispatchesToRegisteredHooksInOrder(t *testing.T) {
	client := newWebhookTestClient(t)
	var calls []string
	client.OnWebhook(func(_ context.Context, event *WebhookEvent) error {
		calls = append(calls, "one")
		if event.GetEvent() != "room_started" || event.GetId() != "evt-1" {
			t.Fatalf("unexpected event: %+v", event)
		}
		return nil
	})
	client.OnWebhook(func(_ context.Context, event *WebhookEvent) error {
		calls = append(calls, "two")
		return nil
	})
	body := `{"event":"room_started","id":"evt-1"}`
	if err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "one" || calls[1] != "two" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
}

func TestReceiveWebhookRejectsBadSignatureWithoutCallingHandlers(t *testing.T) {
	client := newWebhookTestClient(t)
	var calls int
	client.OnWebhook(func(context.Context, *WebhookEvent) error { calls++; return nil })
	body := `{"event":"room_started","id":"evt-2"}`
	// 用错误 key 签名。
	if err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "wrong-key"), "signing-key", nil); err == nil {
		t.Fatal("expected signature error")
	}
	// 篡改 body 后签名不匹配。
	request := signedWebhookRequest(t, body, "signing-key")
	request.Body = http.NoBody
	if err := client.ReceiveWebhook(context.Background(), request, "signing-key", nil); err == nil {
		t.Fatal("expected checksum error")
	}
	if calls != 0 {
		t.Fatalf("handlers must not run on verification failure, calls = %d", calls)
	}
}

func TestReceiveWebhookDeliversUnknownEvent(t *testing.T) {
	client := newWebhookTestClient(t)
	got := make(chan *WebhookEvent, 1)
	client.OnWebhook(func(_ context.Context, event *WebhookEvent) error {
		got <- event
		return nil
	})
	body := `{"event":"unknown_event_xyz","id":"evt-3"}`
	if err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-got:
		if event.GetEvent() != "unknown_event_xyz" || event.GetId() != "evt-3" {
			t.Fatalf("unexpected unknown event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected unknown event delivery")
	}
}

func TestReceiveWebhookOneShotHandlerRunsAfterRegisteredHooks(t *testing.T) {
	client := newWebhookTestClient(t)
	var calls []string
	client.OnWebhook(func(context.Context, *WebhookEvent) error { calls = append(calls, "hook"); return nil })
	body := `{"event":"room_finished","id":"evt-4"}`
	err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key",
		func(context.Context, *WebhookEvent) error { calls = append(calls, "one-shot"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "hook" || calls[1] != "one-shot" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
}

func TestReceiveWebhookAggregatesHandlerErrors(t *testing.T) {
	client := newWebhookTestClient(t)
	hookErr := errors.New("hook failed")
	client.OnWebhook(func(context.Context, *WebhookEvent) error { return hookErr })
	body := `{"event":"room_started","id":"evt-5"}`
	err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil)
	if !errors.Is(err, hookErr) {
		t.Fatalf("expected hook error to propagate, got %v", err)
	}
}

func TestReceiveWebhookRecoversHandlerPanic(t *testing.T) {
	client := newWebhookTestClient(t)
	client.OnWebhook(func(context.Context, *WebhookEvent) error { panic("webhook boom") })
	body := `{"event":"room_started","id":"evt-6"}`
	err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil)
	var panicErr *HookPanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected *HookPanicError, got %T: %v", err, err)
	}
	if panicErr.Value != "webhook boom" {
		t.Fatalf("unexpected panic value: %+v", panicErr)
	}
}

func TestOnWebhookUnsubscribeStopsDelivery(t *testing.T) {
	client := newWebhookTestClient(t)
	var calls int
	sub := client.OnWebhook(func(context.Context, *WebhookEvent) error { calls++; return nil })
	body := `{"event":"room_started","id":"evt-7"}`
	if err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil); err != nil {
		t.Fatal(err)
	}
	sub.Unsubscribe()
	if err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("unsubscribed hook must not run, calls = %d", calls)
	}
}

func TestReceiveWebhookRejectsNilContext(t *testing.T) {
	client, err := New(WithURL("http://127.0.0.1:7880"), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	request, err := http.NewRequest(http.MethodPost, "http://localhost/webhook", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	err = client.ReceiveWebhook(nil, request, "signing-key", func(context.Context, *WebhookEvent) error { return nil })
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid configuration, got %v", err)
	}
}

func TestReceiveWebhookAfterCloseReturnsErrClosed(t *testing.T) {
	client := newWebhookTestClient(t)
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	body := `{"event":"room_started","id":"evt-8"}`
	err := client.ReceiveWebhook(context.Background(), signedWebhookRequest(t, body, "signing-key"), "signing-key", nil)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed after Close, got %v", err)
	}
	// 关闭后注册返回空订阅，不 panic。
	sub := client.OnWebhook(func(context.Context, *WebhookEvent) error { return nil })
	sub.Unsubscribe()
}
