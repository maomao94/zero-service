package gateway

import (
	"net/http"

	"zero-service/app/opengtw/internal/logic/gateway"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/common/gtwx"
)

// 存活探针（不鉴权）
func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := gateway.NewHealthLogic(r.Context(), svcCtx)
		resp, err := l.Health()
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
