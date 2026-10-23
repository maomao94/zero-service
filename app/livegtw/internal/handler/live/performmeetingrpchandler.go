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

// 服务端对参与者执行 RPC
func PerformMeetingRpcHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PerformMeetingRpcRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := live.NewPerformMeetingRpcLogic(r.Context(), svcCtx)
		resp, err := l.PerformMeetingRpc(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
