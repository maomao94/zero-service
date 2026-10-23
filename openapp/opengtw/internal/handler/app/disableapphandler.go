package app

import (
	"net/http"

	"zero-service/common/gtwx"
	"zero-service/openapp/opengtw/internal/logic/app"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 启用或停用应用
func DisableAppHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DisableAppRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := app.NewDisableAppLogic(r.Context(), svcCtx)
		resp, err := l.DisableApp(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
