package webhook

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/lalhook/internal/logic/webhook"
	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"
)

// 别人从当前节点拉流
func OnSubStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OnSubStartRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := webhook.NewOnSubStartLogic(r.Context(), svcCtx)
		resp, err := l.OnSubStart(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
