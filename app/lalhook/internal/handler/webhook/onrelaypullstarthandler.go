package webhook

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/lalhook/internal/logic/webhook"
	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"
)

// 回源拉流成功
func OnRelayPullStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OnRelayPullStartRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := webhook.NewOnRelayPullStartLogic(r.Context(), svcCtx)
		resp, err := l.OnRelayPullStart(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
