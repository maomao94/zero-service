package gtw

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 下载文件
func MfsDownloadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DownloadFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := gtw.NewMfsDownloadFileLogic(r.Context(), svcCtx, r, w)
		err := l.MfsDownloadFile(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			//gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
