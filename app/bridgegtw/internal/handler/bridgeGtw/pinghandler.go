package bridgeGtw

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/bridgegtw/internal/logic/bridgeGtw"
	"zero-service/app/bridgegtw/internal/svc"
)

// ping
func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := bridgeGtw.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
