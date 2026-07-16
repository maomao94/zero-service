package handler

import (
	"context"

	"zero-service/common/gnetx"
	"zero-service/common/isp"
)

// HandleUnimplemented 对所有未实现指令返回 251-3 code=500，对标 Java SipHandlerInterceptor.notSupported。
func HandleUnimplemented(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
	return nil, isp.ErrUnimplemented
}

// HandleFallbackUnimplemented 记录未匹配消息并返回未实现错误，由 wrapper 统一转 251-3 code=500。
func HandleFallbackUnimplemented(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
	isp.LogFallback(ctx, req)
	return nil, isp.ErrUnimplemented
}
