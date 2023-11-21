package gtw

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

func ForwardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ForwardRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := gtw.NewForwardLogic(r.Context(), svcCtx)
		resp, err := l.Forward(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
