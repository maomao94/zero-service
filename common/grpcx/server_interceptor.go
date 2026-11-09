package grpcx

import (
	"context"
	"runtime/debug"

	"zero-service/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

// LoggerInterceptor 一元 RPC 服务端拦截器：
// 从 gRPC incoming metadata 提取上下文字段，注入到 handler 的 context 中；
// 在出口记录错误、恢复 panic，并把未转换的错误归一化为合规 status。
func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	ctx = ExtractFromGrpcMD(ctx)
	method := ""
	if info != nil {
		method = info.FullMethod
	}
	defer func() {
		if r := recover(); r != nil {
			logx.WithContext(ctx).Errorf("rpc panic: %s: %v\n%s", method, r, debug.Stack())
			resp, err = nil, errx.SystemBusyError()
			return
		}
		if err != nil {
			err = errx.ToGRPCStatus(err)
			code, _ := errx.CodeFromStatus(err)
			logx.WithContext(ctx).Errorf("rpc error: method=%s code=%d err=%+v", method, code, err)
		}
	}()
	return handler(ctx, req)
}

// StreamLoggerInterceptor 流式 RPC 服务端拦截器：
// 从 gRPC incoming metadata 提取上下文字段，包装到 stream.Context() 中。
// 解决流式 RPC（如 ChatCompletionStream）中 context values 丢失的问题。
// 出口的错误处理与 panic 恢复策略与一元拦截器一致。
func StreamLoggerInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	ctx := ExtractFromGrpcMD(ss.Context())
	method := ""
	if info != nil {
		method = info.FullMethod
	}
	defer func() {
		if r := recover(); r != nil {
			logx.WithContext(ctx).Errorf("rpc panic: %s: %v\n%s", method, r, debug.Stack())
			err = errx.SystemBusyError()
			return
		}
		if err != nil {
			err = errx.ToGRPCStatus(err)
			code, _ := errx.CodeFromStatus(err)
			logx.WithContext(ctx).Errorf("rpc error: method=%s code=%d err=%+v", method, code, err)
		}
	}()
	return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
}

// wrappedStream 包装 grpc.ServerStream，覆盖 Context() 返回增强后的 ctx。
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
