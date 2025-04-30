package gtw

import (
	"net/http"

	xhttp "github.com/zeromicro/x/http"
	"zero-service/gtw/internal/logic/gtw"
	"zero-service/gtw/internal/svc"
)

// ping
func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := gtw.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping()
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
