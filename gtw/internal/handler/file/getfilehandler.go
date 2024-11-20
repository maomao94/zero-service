package file

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/gtw/internal/logic/file"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 下载文件
func GetFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetFileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := file.NewGetFileLogic(r.Context(), svcCtx)
		err := l.GetFile(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
