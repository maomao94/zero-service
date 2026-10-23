package middleware

import (
	"context"
	"net/http"

	"zero-service/common/gtwx"
	"zero-service/third_party/extproto"

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
	cache *CredentialCache
}

func NewOpenAuthMiddleware(cache *CredentialCache) *OpenAuthMiddleware {
	return &OpenAuthMiddleware{cache: cache}
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

		m.cache.TouchLastUsed(appID)
		next(w, r.WithContext(WithAppID(r.Context(), appID)))
	}
}
