package solo

import (
	"net/http"

	"zero-service/aiapp/aigtw/internal/logic/solo"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SoloChatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := solo.NewChatLogic(r.Context(), svcCtx, w, r)
		err := l.Chat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
		// SSE 流式响应，不需要返回 JSON
	}
}
