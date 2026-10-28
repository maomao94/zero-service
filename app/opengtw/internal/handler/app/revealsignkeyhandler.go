package app

import (
	"net/http"

	"zero-service/app/opengtw/internal/logic/app"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/common/gtwx"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 查看当前签名密钥（审计）
func RevealSignKeyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 管理面已统一挂 NoStore；这里再显式设置一次，避免将来调整路由中间件时
		// 把包含 signKey 的响应暴露给浏览器/代理缓存。
		w.Header().Set("Cache-Control", "no-store")

		var req types.RevealSignKeyRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := app.NewRevealSignKeyLogic(r.Context(), svcCtx)
		resp, err := l.RevealSignKey(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
