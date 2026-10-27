package app

import (
	"net/http"

	"zero-service/app/opengtw/internal/logic/app"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/common/gtwx"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 创建应用（secret 仅返回一次）
func CreateAppHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateAppRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := app.NewCreateAppLogic(r.Context(), svcCtx)
		resp, err := l.CreateApp(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
