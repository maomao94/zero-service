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
		c := svcCtx.Config
		deps := map[string]string{}
		if c.Knowledge.Enabled {
			deps["knowledge_backend"] = c.Knowledge.EffectiveBackend()
		}
		if svcCtx.Knowledge != nil {
			deps["knowledge"] = "ok"
		} else if c.Knowledge.Enabled {
			deps["knowledge"] = "misconfigured"
			if svcCtx.KnowledgeInitErr != "" {
				deps["knowledge_error"] = svcCtx.KnowledgeInitErr
			}
		} else {
			deps["knowledge"] = "disabled"
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"status":       "ok",
			"version":      "aigtw",
			"timestamp":    time.Now().Unix(),
			"dependencies": deps,
		})
	}
}
