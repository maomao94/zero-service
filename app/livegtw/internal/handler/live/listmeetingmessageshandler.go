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

// 查询聊天记录
func ListMeetingMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListMeetingMessagesRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := live.NewListMeetingMessagesLogic(r.Context(), svcCtx)
		resp, err := l.ListMeetingMessages(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
