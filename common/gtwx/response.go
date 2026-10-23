package gtwx

import (
	"context"
	"net/http"
	"strings"

	"zero-service/common/errx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
)

// WriteError 以 HTTP 200 + 统一信封返回错误，永不输出 HTTP 4xx/5xx。
//
// 注意：xhttp.JsonBaseResponseCtx 只负责套信封，不解析 extproto 错误码——
// 它遇到 gRPC status 时取的是 gRPC 数字码（3/5/16），遇到普通 error 时是 -1。
// 因此必须先经 errx.NormalizeError 归一化并校验，再写入信封。
func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	code, msg := errx.NormalizeError(err)
	xhttp.JsonBaseResponseCtx(ctx, w, xerrors.New(code, msg))
}

// WriteParseError 将请求解析错误映射为参数错误码。
func WriteParseError(ctx context.Context, w http.ResponseWriter, err error) {
	WriteError(ctx, w, xerrors.New(int(extproto.Code__1_00_PARAM), err.Error()))
}

// WriteOK 以 HTTP 200 + 统一信封返回成功数据。
func WriteOK(ctx context.Context, w http.ResponseWriter, data any) {
	xhttp.JsonBaseResponseCtx(ctx, w, data)
}

// UnauthorizedHandler 供 go-zero rest.WithUnauthorizedCallback 使用。
// go-zero 会在回调之后补写 401，但 HeaderOnceResponseWriter 只接受第一次写入，
// 因此这里写出的 200 信封会生效。
func UnauthorizedHandler(w http.ResponseWriter, r *http.Request, _ error) {
	ensureRequestID(w, r)
	code := int(extproto.Code__1_00_UNAUTHORIZED)
	WriteError(r.Context(), w, xerrors.New(code, errx.ErrorMessage(code)))
}

// NotFoundHandler 统一处理未匹配路由。
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	ensureRequestID(w, r)
	code := int(extproto.Code__1_00_ROUTE_NOT_FOUND)
	WriteError(r.Context(), w, xerrors.New(code, errx.ErrorMessage(code)))
}

// MethodNotAllowedHandler 统一处理请求方法不允许。
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	ensureRequestID(w, r)
	code := int(extproto.Code__1_00_METHOD_NOT_ALLOWED)
	WriteError(r.Context(), w, xerrors.New(code, errx.ErrorMessage(code)))
}

// ensureRequestID 为绕过业务中间件的兜底响应补上请求 ID。
func ensureRequestID(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(w.Header().Get("X-Request-Id")) != "" {
		return
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		id, err := tool.SimpleUUID()
		if err != nil {
			return
		}
		requestID = id
	}
	w.Header().Set("X-Request-Id", requestID)
}
