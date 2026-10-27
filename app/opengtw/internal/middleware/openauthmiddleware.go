package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/common/gtwx"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/stores/redis"
	xerrors "github.com/zeromicro/x/errors"
)

type appIDKey struct{}

// WithAppID 将应用 ID 写入请求上下文，供转发与日志使用。
func WithAppID(ctx context.Context, appID string) context.Context {
	return context.WithValue(ctx, appIDKey{}, appID)
}

// AppIDFromContext 读取 Basic 鉴权通过后的应用 ID。
func AppIDFromContext(ctx context.Context) string {
	appID, _ := ctx.Value(appIDKey{}).(string)
	return appID
}

// OpenAuthMiddleware 校验 Authorization: Basic base64(appId:secret)。
type OpenAuthMiddleware struct {
	cache         *CredentialCache
	encryptionKey string
	nonceRedis    *redis.Redis
}

func NewOpenAuthMiddleware(cache *CredentialCache, encryptionKey string, nonceRedis *redis.Redis) *OpenAuthMiddleware {
	return &OpenAuthMiddleware{cache: cache, encryptionKey: encryptionKey, nonceRedis: nonceRedis}
}

func (m *OpenAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, secret, ok := r.BasicAuth()
		if !ok || appID == "" || secret == "" {
			gtwx.WriteError(r.Context(), w, xerrors.New(
				int(extproto.Code__1_30_APP_SECRET_INVALID), "缺少或非法的 Basic 凭据"))
			return
		}

		cred, ok := m.cache.Get(appID)
		if !ok {
			// 不存在与已停用统一返回同一错误码，避免泄露应用状态。
			code := int(extproto.Code__1_30_APP_NOT_FOUND)
			gtwx.WriteError(r.Context(), w, xerrors.New(code, errorMessage(code)))
			return
		}
		if !cred.Match(secret) {
			code := int(extproto.Code__1_30_APP_SECRET_INVALID)
			gtwx.WriteError(r.Context(), w, xerrors.New(code, errorMessage(code)))
			return
		}
		if cred.SignRequired {
			if err := m.verifySignature(r, cred); err != nil {
				gtwx.WriteError(r.Context(), w, xerrors.New(int(extproto.Code__1_30_SIGNATURE_INVALID), "签名无效"))
				return
			}
		}

		m.cache.TouchLastUsed(appID)
		next(w, r.WithContext(WithAppID(r.Context(), appID)))
	}
}

// verifySignature 完成“读取body → 解析签名要素 → 校验时间窗 → 校验 HMAC → 占用 nonce”。
//
// 解析、canonical string 和 HMAC 校验属于跨语言签名协议，统一放在 security 包；
// 这里只处理 HTTP body、依赖检查和 Redis 防重放。
func (m *OpenAuthMiddleware) verifySignature(r *http.Request, cred *Credential) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read signed request: %w", err)
	}
	// 读取后必须还原 body：后续 GatewayHandler 还要拿原始 JSON 提取 data 并转发。
	r.Body = io.NopCloser(bytes.NewReader(body))

	req, err := security.ParseSignedRequest(body)
	if err != nil {
		return err
	}
	if err := req.Validate(cred.AppId, time.Now()); err != nil {
		return err
	}
	if m.nonceRedis == nil || m.encryptionKey == "" {
		return fmt.Errorf("signature dependencies not configured")
	}
	if err := req.Verify(cred.AppId, r.URL.Path, cred.SignKeys(m.encryptionKey)); err != nil {
		return err
	}

	// 签名通过后再占用 nonce；TTL 取两倍时间窗，确保窗口内的重复请求都能命中。
	created, err := m.nonceRedis.SetnxExCtx(
		r.Context(),
		"openapi:nonce:"+cred.AppId+":"+req.Nonce,
		"1",
		int(2*security.SignatureTimeWindow.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("reserve nonce: %w", err)
	}
	if !created {
		return fmt.Errorf("nonce already used")
	}
	return nil
}
