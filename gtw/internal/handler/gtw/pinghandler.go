package gtw

import (
	"net/http"

	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
)

// ping
func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := gtw.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping()
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
