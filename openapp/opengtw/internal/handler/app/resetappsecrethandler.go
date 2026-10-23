package app

import (
	"net/http"

	"zero-service/common/gtwx"
	"zero-service/openapp/opengtw/internal/logic/app"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 重置应用密钥（旧密钥保留 24h）
func ResetAppSecretHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetAppSecretRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := app.NewResetAppSecretLogic(r.Context(), svcCtx)
		resp, err := l.ResetAppSecret(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
