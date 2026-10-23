package file

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/file"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 获取文件信息
func StatFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.StatFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := file.NewStatFileLogic(r.Context(), svcCtx)
		resp, err := l.StatFile(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
