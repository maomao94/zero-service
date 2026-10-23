package gtw

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// forward
func ForwardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ForwardRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := gtw.NewForwardLogic(r.Context(), svcCtx)
		resp, err := l.Forward(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
