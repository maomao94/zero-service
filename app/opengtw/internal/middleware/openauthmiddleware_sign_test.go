package middleware

import (
	"bytes"
	"context"
	"encoding/json"
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

// TestOpenAuthMiddlewareSignature 覆盖签名应用的 canonical string、nonce 防重放与失败码。
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

	send := func(nonce, key string, timestamp int64) bool {
		t.Helper()
		data := `{"limit":1}`
		material := &security.SignedRequest{
			AppID:     "app_sign",
			Method:    "pod.list",
			Timestamp: timestamp,
			Nonce:     nonce,
			TID:       "t-1",
			Data:      json.RawMessage(data),
		}
		body, err := json.Marshal(map[string]any{
			"appId":     material.AppID,
			"method":    material.Method,
			"timestamp": material.Timestamp,
			"nonce":     material.Nonce,
			"tid":       material.TID,
			"data":      material.Data,
			"sign":      material.ComputeSign("app_sign", security.SignatureGatewayPath, key),
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}

		called := false
		handler := mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			forwarded, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read forwarded body: %v", err)
			}
			if !bytes.Equal(forwarded, body) {
				t.Errorf("forwarded body = %s, want %s", forwarded, body)
			}
			called = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodPost, "/v1/gateway", bytes.NewReader(body))
		req.SetBasicAuth("app_sign", "sec_sign")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if called {
			return true
		}

		var envelope xhttp.BaseResponse[any]
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode body %q: %v", rec.Body.String(), err)
		}
		if envelope.Code != int(extproto.Code__1_30_SIGNATURE_INVALID) {
			t.Fatalf("business code = %d, want %d", envelope.Code, extproto.Code__1_30_SIGNATURE_INVALID)
		}
		return false
	}

	now := time.Now().UnixMilli()
	if !send("nonce-ok", signKey, now) {
		t.Fatal("valid signature should pass")
	}
	if send("nonce-ok", signKey, now) {
		t.Fatal("replayed nonce should be rejected")
	}
	if send("nonce-bad-key", "sk_wrong", now) {
		t.Fatal("wrong sign key should be rejected")
	}
	if send("nonce-stale", signKey, now-6*60*1000) {
		t.Fatal("timestamp outside 5 minutes should be rejected")
	}
}
