// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ssegtw

import (
	"net/http"

	"zero-service/aiapp/ssegtw/internal/logic/ssegtw"
	"zero-service/aiapp/ssegtw/internal/svc"
	"zero-service/common/gtwx"
)

// ping
func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := ssegtw.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping()
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
