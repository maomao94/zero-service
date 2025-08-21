package api

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/lalhook/internal/logic/api"
	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"
)

// 查询 TS 文件列表（按时间区间）
func ListTsFilesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ApiListTsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := api.NewListTsFilesLogic(r.Context(), svcCtx)
		resp, err := l.ListTsFiles(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
