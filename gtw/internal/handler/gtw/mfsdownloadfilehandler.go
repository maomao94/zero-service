package gtw

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 下载文件
func MfsDownloadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DownloadFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := gtw.NewMfsDownloadFileLogic(r.Context(), svcCtx, r, w)
		err := l.MfsDownloadFile(&req)
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			//xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
