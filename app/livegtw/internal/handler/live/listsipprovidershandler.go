// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package live

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/livegtw/internal/logic/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/gtwx"
)

// 查询 SIP 供应商列表
func ListSipProvidersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListSipProvidersRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := live.NewListSipProvidersLogic(r.Context(), svcCtx)
		resp, err := l.ListSipProviders(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
