package webhook

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
)

// errWebhook 模拟 WebhookNotify RPC 转发失败。
var errWebhook = errors.New("rpc unavailable")

// signBody 生成 LiveKit webhook 验签 token（body sha256 的 base64 进 JWT Sha256 claim，
// 与 livekit protocol webhook 验签逻辑一致）。
func signBody(body string, secret string) string {
	hash := sha256.Sum256([]byte(body))
	token, _ := auth.NewAccessToken("api-key", secret).
		SetSha256(base64.StdEncoding.EncodeToString(hash[:])).
		SetValidFor(time.Hour).
		ToJWT()
	return token
}

func TestLiveKitWebhookHandlerInvalidSignature(t *testing.T) {
	svcCtx := newTestSvcCtx(&fakeLiveRpcCli{})
	handler := LiveKitWebhookHandler(svcCtx)

	// 无 Authorization：验签失败 → 401，不调用业务
	req := newWebhookRequest("{}", "")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
	if svcCtx.LiveRpcCli.(*fakeLiveRpcCli).webhookCalled {
		t.Fatal("business should not be called on invalid signature")
	}
}

func TestLiveKitWebhookHandlerValidSignatureForwards(t *testing.T) {
	svcCtx := newTestSvcCtx(&fakeLiveRpcCli{})
	handler := LiveKitWebhookHandler(svcCtx)

	// 正确签名：验签通过 → 200 且转发 WebhookNotify
	body := `{"id":"EVT-1","event":"participant_joined","room":{"name":"M001"},"participant":{"identity":"bob","name":"Bob"}}`
	token := signBody(body, "secret")
	req := newWebhookRequest(body, token)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	fake := svcCtx.LiveRpcCli.(*fakeLiveRpcCli)
	if !fake.webhookCalled {
		t.Fatal("WebhookNotify should be forwarded")
	}
	if len(fake.webhookData) == 0 {
		t.Fatal("webhook data should not be empty")
	}
}

func TestLiveKitWebhookHandlerForwardErrorReturns500(t *testing.T) {
	svcCtx := newTestSvcCtx(&fakeLiveRpcCli{webhookErr: errWebhook})
	handler := LiveKitWebhookHandler(svcCtx)

	// 签名正确但后续 RPC 转发失败：应返回 500 让 LiveKit 重推（不返回 200 丢事件）
	body := `{"id":"EVT-3","event":"room_finished","room":{"name":"M001"}}`
	token := signBody(body, "secret")
	req := newWebhookRequest(body, token)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !svcCtx.LiveRpcCli.(*fakeLiveRpcCli).webhookCalled {
		t.Fatal("WebhookNotify should still be forwarded on valid signature")
	}
}

func TestLiveKitWebhookHandlerWrongSecretRejected(t *testing.T) {
	svcCtx := newTestSvcCtx(&fakeLiveRpcCli{})
	handler := LiveKitWebhookHandler(svcCtx)

	// 用错误 secret 签名：验签失败 → 401
	body := `{"id":"EVT-2","event":"participant_left","room":{"name":"M001"}}`
	token := signBody(body, "wrong-secret")
	req := newWebhookRequest(body, token)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
	if svcCtx.LiveRpcCli.(*fakeLiveRpcCli).webhookCalled {
		t.Fatal("business should not be called on wrong secret")
	}
}
