package handler

import (
	"net/http"
	"time"

	"zero-service/aiapp/aigtw/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// HealthHandler 返回与 aisolo Health RPC 对齐的依赖摘要，便于网关探针与运维对照 aisolo。
func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps := svcCtx.Dependencies()
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"status":       "ok",
			"ready":        svcCtx.Ready(),
			"version":      "aigtw",
			"timestamp":    time.Now().Unix(),
			"dependencies": deps,
		})
	}
}
