package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
	xhttp "github.com/zeromicro/x/http"
)

const testSignKeyEncryptionKey = "0123456789abcdef0123456789abcdef"

// TestOpenAuthMiddlewareSignature 覆盖 X-Sign 头、bodyHash、nonce 防重放与失败码。
func TestOpenAuthMiddlewareSignature(t *testing.T) {
	mr := miniredis.RunT(t)
	db, appStore, secretStore := openAuthTestDB(t)

	signKey, err := security.NewSignKey()
	if err != nil {
		t.Fatalf("generate sign key: %v", err)
	}
	cipher, err := security.EncryptSignKey(testSignKeyEncryptionKey, signKey)
	if err != nil {
		t.Fatalf("encrypt sign key: %v", err)
	}
	seedApp(t, db, "app_sign", "signed", 1,
		&gormmodel.OpenAppSecret{
			SecretHash:    security.HashSecret("sec_sign"),
			SignKeyCipher: cipher,
			Status:        1,
		},
	)
	if err := db.Model(&gormmodel.OpenApp{}).Where("app_id = ?", "app_sign").
		Update("sign_required", true).Error; err != nil {
		t.Fatalf("enable signing: %v", err)
	}

	cache := NewCredentialCache(appStore, secretStore, time.Hour)
	cache.MustRefresh(context.Background())

	mw := NewOpenAuthMiddleware(cache, testSignKeyEncryptionKey, redis.New(mr.Addr()))

	// timestamp、nonce 放在 body 里，由 bodyHash 覆盖；签名头只放 sign。
	signedBody := func(timestamp int64, nonce string) []byte {
		return []byte(fmt.Sprintf(
			`{"method":"pod.list","data":{"limit":1},"tid":"t-1","timestamp":%d,"nonce":%q}`,
			timestamp, nonce,
		))
	}

	build := func(nonce, key string, timestamp int64) (*http.Request, []byte) {
		requestBody := signedBody(timestamp, nonce)
		input := &security.SignatureInput{
			AppID:     "app_sign",
			Timestamp: timestamp,
			Nonce:     nonce,
			Path:      security.SignatureGatewayPath,
			BodyHash:  security.HashBody(requestBody),
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/gateway", bytes.NewReader(requestBody))
		req.SetBasicAuth("app_sign", "sec_sign")
		req.Header.Set(HeaderSign, security.SignatureVersion+":"+input.ComputeSign(key))
		return req, requestBody
	}

	send := func(req *http.Request, wantBody []byte) (bool, int) {
		called := false
		handler := mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			forwarded, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read forwarded body: %v", err)
			}
			if !bytes.Equal(forwarded, wantBody) {
				t.Errorf("forwarded body = %s, want %s", forwarded, wantBody)
			}
			called = true
			w.WriteHeader(http.StatusOK)
		})
		rec := httptest.NewRecorder()
		handler(rec, req)
		if called {
			return true, 0
		}
		var envelope xhttp.BaseResponse[any]
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode body %q: %v", rec.Body.String(), err)
		}
		return false, envelope.Code
	}

	now := time.Now().UnixMilli()

	req, body := build("nonce-ok", signKey, now)
	if called, code := send(req, body); !called || code != 0 {
		t.Fatalf("valid signature should pass, called=%v code=%d", called, code)
	}
	req, body = build("nonce-ok", signKey, now)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("replayed nonce should be rejected, called=%v code=%d", called, code)
	}
	req, body = build("nonce-bad-key", "sk_wrong", now)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("wrong sign key should be rejected, called=%v code=%d", called, code)
	}
	req, body = build("nonce-stale", signKey, now-6*60*1000)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("timestamp outside 5 minutes should be rejected, called=%v code=%d", called, code)
	}

	// 用 body 的签名发送被篡改的 body，bodyHash 必须不匹配。
	req, _ = build("nonce-tampered", signKey, now)
	tampered := []byte(`{"method":"pod.delete","data":{"limit":1},"tid":"t-1","timestamp":1,"nonce":"nonce-tampered"}`)
	req.Body = io.NopCloser(bytes.NewReader(tampered))
	req.ContentLength = int64(len(tampered))
	if called, code := send(req, tampered); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("tampered body should be rejected, called=%v code=%d", called, code)
	}

	// 缺少 X-Sign 头。
	req, body = build("nonce-missing-header", signKey, now)
	req.Header.Del(HeaderSign)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("missing signature header should be rejected, called=%v code=%d", called, code)
	}

	// 不支持的协议版本。
	req, body = build("nonce-version", signKey, now)
	_, sign, _ := security.ParseSignHeader(req.Header.Get(HeaderSign))
	req.Header.Set(HeaderSign, "v2:"+sign)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("unsupported sign version should be rejected, called=%v code=%d", called, code)
	}

	// nonce 字符集非法。
	req, body = build("nonce with space", signKey, now)
	if called, code := send(req, body); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("invalid nonce should be rejected, called=%v code=%d", called, code)
	}

	// body 缺少 timestamp/nonce：签名本身合法，但防重放要素缺失。
	withoutMeta := []byte(`{"method":"pod.list","data":{"limit":1},"tid":"t-1"}`)
	input := &security.SignatureInput{
		AppID:    "app_sign",
		Path:     security.SignatureGatewayPath,
		BodyHash: security.HashBody(withoutMeta),
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/gateway", bytes.NewReader(withoutMeta))
	req.SetBasicAuth("app_sign", "sec_sign")
	req.Header.Set(HeaderSign, security.SignatureVersion+":"+input.ComputeSign(signKey))
	if called, code := send(req, withoutMeta); called || code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
		t.Fatalf("missing timestamp/nonce should be rejected, called=%v code=%d", called, code)
	}
}
