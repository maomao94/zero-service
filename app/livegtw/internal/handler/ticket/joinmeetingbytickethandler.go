// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ticket

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
	"zero-service/app/livegtw/internal/logic/ticket"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
)

// 根据票据加入会议（无需JWT）
func JoinMeetingByTicketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.JoinMeetingByTicketRequest
		if err := httpx.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}

		l := ticket.NewJoinMeetingByTicketLogic(r.Context(), svcCtx)
		resp, err := l.JoinMeetingByTicket(&req)
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
