package gateway

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"zero-service/common/gtwx"
	"zero-service/openapp/opengtw/internal/logic/gateway"
	"zero-service/openapp/opengtw/internal/middleware"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

// 统一转发（Basic 鉴权，method 决定下游调用）
func GatewayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var req types.GatewayRequest
		// data 必须是 json.RawMessage，go-zero mapping 不支持原样透传，这里直接用标准库解码。
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		if req.Tid == "" {
			req.Tid = middleware.RequestIDFromContext(r.Context())
		}
		if req.Tid != "" {
			w.Header().Set("X-Request-Id", req.Tid)
		}

		l := gateway.NewGatewayLogic(r.Context(), svcCtx)
		data, err := l.Gateway(&req)
		code := 0
		if err != nil {
			var codeMsg *xerrors.CodeMsg
			if errors.As(err, &codeMsg) && codeMsg != nil {
				code = codeMsg.Code
			} else {
				code = 100998
			}
			logGatewayRequest(r, req, code, start)
			gtwx.WriteError(r.Context(), w, err)
			return
		}

		logGatewayRequest(r, req, code, start)
		gtwx.WriteOK(r.Context(), w, data)
	}
}

func logGatewayRequest(r *http.Request, req types.GatewayRequest, code int, start time.Time) {
	logx.WithContext(r.Context()).Infow("openapi request",
		logx.Field("app", middleware.AppIDFromContext(r.Context())),
		logx.Field("method", req.Method),
		logx.Field("ip", clientIP(r)),
		logx.Field("code", code),
		logx.Field("duration", time.Since(start).String()),
		logx.Field("requestId", req.Tid),
	)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if idx := strings.Index(forwarded, ","); idx >= 0 {
			forwarded = forwarded[:idx]
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
