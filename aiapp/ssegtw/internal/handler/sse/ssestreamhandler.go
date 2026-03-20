// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package sse

import (
	"net/http"

	"zero-service/aiapp/ssegtw/internal/logic/sse"
	"zero-service/aiapp/ssegtw/internal/svc"
	"zero-service/aiapp/ssegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// SSE事件流
func SseStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SSEStreamRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := sse.NewSseStreamLogic(r.Context(), svcCtx, w, r)
		err := l.SseStream(&req)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("sse stream error: %v", err)
		}
	}
}
