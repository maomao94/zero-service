package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"zero-service/app/oryxgtw/internal/config"
	"zero-service/app/oryxgtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// NewOryxHookAuthMiddleware 返回 Oryx 回调凭证校验中间件。
// 校验请求体中的 opaque 字段（Oryx 回调透传的凭证）是否与配置凭证一致，不一致则拒绝回调。
func NewOryxHookAuthMiddleware(c config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if c.HookOpaque == "" {
				next(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			r.Body.Close()

			var req types.HookRequest
			if err := json.Unmarshal(body, &req); err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}

			if req.Opaque != c.HookOpaque {
				httpx.OkJsonCtx(r.Context(), w, types.HookReply{Code: 1})
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next(w, r)
		}
	}
}
