package app

import (
	"net/http"

	"zero-service/app/opengtw/internal/logic/app"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/common/gtwx"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 应用列表
func ListAppsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListAppsRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := app.NewListAppsLogic(r.Context(), svcCtx)
		resp, err := l.ListApps(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
