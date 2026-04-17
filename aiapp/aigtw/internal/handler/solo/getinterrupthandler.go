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

// 获取中断详情（页面刷新后回填 UI）
func GetInterruptHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SoloGetInterruptRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := solo.NewGetInterruptLogic(r.Context(), svcCtx)
		resp, err := l.GetInterrupt(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
