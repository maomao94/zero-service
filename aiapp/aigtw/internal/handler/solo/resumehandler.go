// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package solo

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/solo"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
)

// 中断恢复
func ResumeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SoloInterruptReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := solo.NewResumeLogic(r.Context(), svcCtx)
		resp, err := l.Resume(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
