package gtwx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"zero-service/common/errx"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
)

type ctxKey struct{}

type reqMeta struct {
	Method  string
	Path    string
	StartAt time.Time
}

// RequestLogMiddleware stores request method, path, and start time in context
// for use by the ok/error response handlers.
func RequestLogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meta := &reqMeta{
			Method:  r.Method,
			Path:    r.URL.Path,
			StartAt: time.Now(),
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, meta)))
	}
}

func getReqMeta(ctx context.Context) *reqMeta {
	m, _ := ctx.Value(ctxKey{}).(*reqMeta)
	return m
}

// SetUnifiedErrorHandler 让 go-zero 的全局错误出口也返回 HTTP 200 + 统一信封。
// 仅用于兜底框架级错误（路由、解析、中间件）；业务 handler 应显式调用 WriteError。
func SetUnifiedErrorHandler() {
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		code, msg := errx.NormalizeError(err)
		return http.StatusOK, xhttp.BaseResponse[any]{Code: code, Msg: msg}
	})
}

// SetLogOkHandler sets a global ok handler that logs error responses.
func SetLogOkHandler() {
	httpx.SetOkHandler(func(ctx context.Context, v any) any {
		if m := getReqMeta(ctx); m != nil {
			if resp, ok := v.(xhttp.BaseResponse[any]); ok && resp.Code != xhttp.BusinessCodeOK {
				logc.Error(ctx, fmt.Sprintf("[HTTP] [%d] - %s %s - %s - %s",
					resp.Code, m.Method, m.Path, time.Since(m.StartAt).String(), resp.Msg))
			}
		}
		return v
	})
}
