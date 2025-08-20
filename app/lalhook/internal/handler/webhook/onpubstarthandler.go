package webhook

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/lalhook/internal/logic/webhook"
	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"
)

// 别人推流到当前节点
func OnPubStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OnPubStartRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := webhook.NewOnPubStartLogic(r.Context(), svcCtx)
		resp, err := l.OnPubStart(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
