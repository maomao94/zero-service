// Custom SSE handler (NDJSON over SSE). DO NOT re-generate.
package solo

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/solo"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
)

// ChatHandler 对话 (SSE 流式输出 Solo Protocol 事件).
// 每一帧是完整的 JSON Event, 以 "data: <json>\n\n" 格式推送, 由前端 SSE 解析.
func ChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SoloChatRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// 关键: 告诉反向代理 (Nginx/Envoy) 不要缓冲此响应,
		// 否则 token 会被攒到 flush buffer 满才一次性推给前端。
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		l := solo.NewChatLogic(r.Context(), svcCtx)
		if err := l.Chat(&req, w); err != nil {
			logc.Errorw(r.Context(), "ChatHandler stream error", logc.Field("error", err))
		}
	}
}
