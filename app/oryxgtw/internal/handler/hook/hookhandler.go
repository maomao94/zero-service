// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package hook

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/oryxgtw/internal/logic/hook"
	"zero-service/app/oryxgtw/internal/svc"
	"zero-service/app/oryxgtw/internal/types"
)

// Oryx HTTP 回调统一入口（按 action 字段分发）
func HookHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HookRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := hook.NewHookLogic(r.Context(), svcCtx)
		resp, err := l.Hook(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
