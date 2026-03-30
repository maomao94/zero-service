// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package pass

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/pass"
	"zero-service/aiapp/aigtw/internal/svc"
)

// 获取异步结果统计信息
func AsyncResultStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := pass.NewAsyncResultStatsLogic(r.Context(), svcCtx)
		resp, err := l.AsyncResultStats()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
