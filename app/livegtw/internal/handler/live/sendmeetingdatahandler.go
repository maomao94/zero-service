// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package live

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/app/livegtw/internal/logic/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/gtwx"
)

// 向会议发送 Data（广播/定向）
func SendMeetingDataHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendMeetingDataRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := live.NewSendMeetingDataLogic(r.Context(), svcCtx)
		err := l.SendMeetingData(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, nil)
		}
	}
}
