package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/common/gtwx"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	xerrors "github.com/zeromicro/x/errors"
)

// HeaderSign 是唯一的签名头，格式为 "v1:<hex>"。
// timestamp、nonce 放在 body 中，由 bodyHash 覆盖；body 保持 method/data/tid 等业务字段。
const HeaderSign = "X-Sign"

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
			if err := m.verifySignature(r, cred, appID); err != nil {
				// 对外统一 130105，真实原因只写服务端日志，避免泄露时间窗/nonce/key 细节。
				logx.WithContext(r.Context()).Errorf("openapi signature verify failed app_id=%s: %v", appID, err)
				gtwx.WriteError(r.Context(), w, xerrors.New(int(extproto.Code__1_30_SIGNATURE_INVALID), "签名无效"))
				return
			}
		}

		m.cache.TouchLastUsed(appID)
		next(w, r.WithContext(WithAppID(r.Context(), appID)))
	}
}

// verifySignature 完成“解析 X-Sign → 读取原始 body → 解析 timestamp/nonce → 校验时间窗 → 校验 HMAC → 占用 nonce”。
//
// 解析、canonical string 和 HMAC 校验属于跨语言签名协议，统一放在 security 包；
// 这里只处理 HTTP header/body、依赖检查和 Redis 防重放。
func (m *OpenAuthMiddleware) verifySignature(r *http.Request, cred *Credential, appID string) error {
	if m.nonceRedis == nil || m.encryptionKey == "" {
		return fmt.Errorf("signature dependencies not configured")
	}
	version, sign, err := security.ParseSignHeader(r.Header.Get(HeaderSign))
	if err != nil {
		return err
	}
	if version != security.SignatureVersion {
		return fmt.Errorf("%w: %s", security.ErrSignatureVersion, version)
	}

	// 签名覆盖原始 body 字节：读取后必须还原，GatewayHandler 还要解析并转发。
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read signed request: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	// timestamp、nonce 在 body 里，已由 bodyHash 覆盖；这里只取出用于时间窗和防重放。
	var meta struct {
		Timestamp int64  `json:"timestamp"`
		Nonce     string `json:"nonce"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return fmt.Errorf("parse signed request body: %w", err)
	}

	input := &security.SignatureInput{
		AppID:     appID,
		Timestamp: meta.Timestamp,
		Nonce:     meta.Nonce,
		Path:      r.URL.Path,
		BodyHash:  security.HashBody(body),
		Sign:      sign,
	}
	if err := input.Validate(time.Now()); err != nil {
		return err
	}
	if err := input.Verify(cred.SignKeys(m.encryptionKey)); err != nil {
		return err
	}

	// 签名通过后再占用 nonce；TTL 取两倍时间窗，确保窗口内的重复请求都能命中。
	created, err := m.nonceRedis.SetnxExCtx(
		r.Context(),
		"openapi:nonce:"+appID+":"+input.Nonce,
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
