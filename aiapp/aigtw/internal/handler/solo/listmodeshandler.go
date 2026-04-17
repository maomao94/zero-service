// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package solo

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/solo"
	"zero-service/aiapp/aigtw/internal/svc"
)

// 列出所有 Mode (用户挑 Mode, 不再挑 Agent)
func ListModesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := solo.NewListModesLogic(r.Context(), svcCtx)
		resp, err := l.ListModes()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
