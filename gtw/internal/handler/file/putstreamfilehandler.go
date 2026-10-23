package file

import (
	"net/http"

	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/file"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 上传快文件-grpc单向流
func PutStreamFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PutFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := file.NewPutStreamFileLogic(r.Context(), svcCtx, r)
		resp, err := l.PutStreamFile(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
