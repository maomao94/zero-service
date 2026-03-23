package ai

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/ai"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
)

func ChatCompletionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatCompletionRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ai.NewChatCompletionsLogic(r.Context(), svcCtx, w, r)
		resp, err := l.ChatCompletions(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else if resp != nil {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
		// resp == nil 表示流式响应，Logic 已直接写 SSE
	}
}
