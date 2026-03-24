package interceptor

import (
	"context"

	"zero-service/common/ctxprop"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

// LoggerInterceptor 一元 RPC 服务端拦截器：
// 从 gRPC incoming metadata 提取上下文字段，注入到 handler 的 context 中。
func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	ctx = ctxprop.ExtractFromGrpcMD(ctx)
	resp, err = handler(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
	}
	return resp, err
}

// StreamLoggerInterceptor 流式 RPC 服务端拦截器：
// 从 gRPC incoming metadata 提取上下文字段，包装到 stream.Context() 中。
// 解决流式 RPC（如 ChatCompletionStream）中 context values 丢失的问题。
func StreamLoggerInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ctxprop.ExtractFromGrpcMD(ss.Context())
	err := handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	if err != nil {
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
	}
	return err
}

// wrappedStream 包装 grpc.ServerStream，覆盖 Context() 返回增强后的 ctx。
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
