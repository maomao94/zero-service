// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package pass

import (
	"net/http"

	"zero-service/aiapp/aigtw/internal/logic/pass"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListAsyncResultsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := pass.NewListAsyncResultsLogic(r.Context(), svcCtx)
		var req types.ListAsyncResultsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := l.ListAsyncResults(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
