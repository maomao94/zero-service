// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package meeting

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/livegtw/internal/logic/meeting"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	xhttp "github.com/zeromicro/x/http"
)

// 向会议发送 Data（广播/定向）
func SendMeetingDataHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendMeetingDataReq
		if err := httpx.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}

		l := meeting.NewSendMeetingDataLogic(r.Context(), svcCtx)
		err := l.SendMeetingData(&req)
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			xhttp.JsonBaseResponseCtx(r.Context(), w, nil)
		}
	}
}
