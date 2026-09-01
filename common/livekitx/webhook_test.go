package livekitx

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
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

// receiveWebhook 按业务侧约定调用 SDK webhook.ReceiveWebhookEvent，
// 与文档示例保持一致。
func receiveWebhook(t *testing.T, request *http.Request, signingKey string) (*livekit.WebhookEvent, error) {
	t.Helper()
	return webhook.ReceiveWebhookEvent(request, NewWebhookKeyProvider(signingKey))
}

// TestWebhookKeyProviderAcceptsCorrectSigningKey 验证正确 signing key
// 验签通过，事件字段完整保留。
func TestWebhookKeyProviderAcceptsCorrectSigningKey(t *testing.T) {
	body := `{"event":"room_started","id":"evt-1"}`
	event, err := receiveWebhook(t, signedWebhookRequest(t, body, "signing-key"), "signing-key")
	if err != nil {
		t.Fatal(err)
	}
	if event.GetEvent() != "room_started" || event.GetId() != "evt-1" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

// TestWebhookKeyProviderRejectsWrongSigningKey 验证错误 signing key 验签
// 失败并返回签名错误，不产出任何事件。
func TestWebhookKeyProviderRejectsWrongSigningKey(t *testing.T) {
	body := `{"event":"room_started","id":"evt-2"}`
	// 请求用 wrong-key 签名，业务用 signing-key 校验。
	request := signedWebhookRequest(t, body, "wrong-key")
	if event, err := receiveWebhook(t, request, "signing-key"); err == nil {
		t.Fatalf("expected signature error, got event %+v", event)
	}
	// 请求用 signing-key 签名，业务用 wrong-key 校验。
	request = signedWebhookRequest(t, body, "signing-key")
	if event, err := receiveWebhook(t, request, "wrong-key"); err == nil {
		t.Fatalf("expected signature error, got event %+v", event)
	}
}

// TestWebhookKeyProviderRejectsTamperedBody 验证篡改 body 后摘要不匹配，
// 验签失败。
func TestWebhookKeyProviderRejectsTamperedBody(t *testing.T) {
	body := `{"event":"room_started","id":"evt-3"}`
	request := signedWebhookRequest(t, body, "signing-key")
	request.Body = http.NoBody
	if event, err := receiveWebhook(t, request, "signing-key"); err == nil {
		t.Fatalf("expected checksum error, got event %+v", event)
	}
}

// TestWebhookKeyProviderDeliversUnknownEvent 验证未知事件类型验签通过后
// 安全交付，保留事件 ID/type 由业务决定处理策略。
func TestWebhookKeyProviderDeliversUnknownEvent(t *testing.T) {
	body := `{"event":"unknown_event_xyz","id":"evt-4"}`
	event, err := receiveWebhook(t, signedWebhookRequest(t, body, "signing-key"), "signing-key")
	if err != nil {
		t.Fatal(err)
	}
	if event.GetEvent() != "unknown_event_xyz" || event.GetId() != "evt-4" {
		t.Fatalf("unexpected unknown event: %+v", event)
	}
}

// TestWebhookKeyProviderRejectsEmptySigningKey 验证空 signing key 验签
// 失败（KeyProvider 不持有任何可匹配的 secret）。
func TestWebhookKeyProviderRejectsEmptySigningKey(t *testing.T) {
	body := `{"event":"room_started","id":"evt-5"}`
	request := signedWebhookRequest(t, body, "signing-key")
	if event, err := receiveWebhook(t, request, ""); err == nil {
		t.Fatalf("expected signature error for empty key, got event %+v", event)
	}
}

// TestNewWebhookKeyProviderSatisfiesKeyProvider 验证返回值实现
// auth.KeyProvider 接口且对任意 key claim 返回同一个 signing key。
func TestNewWebhookKeyProviderSatisfiesKeyProvider(t *testing.T) {
	provider := NewWebhookKeyProvider("signing-key")
	if provider.NumKeys() != 1 {
		t.Fatalf("NumKeys() = %d, want 1", provider.NumKeys())
	}
	for _, claim := range []string{"devkey", "", "any-key-claim"} {
		if got := provider.GetSecret(claim); got != "signing-key" {
			t.Fatalf("GetSecret(%q) = %q, want signing-key", claim, got)
		}
	}
}
