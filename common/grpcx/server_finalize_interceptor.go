package grpcx

import (
	"context"
	"runtime/debug"

	"zero-service/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

// ServerFinalizeInterceptor 是服务端一元 RPC 的最外层拦截器，必须注册在 go-zero
// 内置拦截器（含 UnaryTimeoutInterceptor）之外。
//
// 内层 LoggerInterceptor 位于超时拦截器之内，超时/限流/熔断会在到达它之前返回，
// 客户端拿不到 extproto detail。这一层只做出口归一化：
//   - 已有 detail 的错误原样保留；
//   - 没有 detail 的错误（如超时的 DeadlineExceeded）补一条默认 detail，
//     保证客户端始终能按六位码处理。
//
// 同时兜底 panic 并注入上下文字段；错误日志仍由内层 LoggerInterceptor 负责。
func ServerFinalizeInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp any, err error) {
	ctx = ExtractFromGrpcMD(ctx)
	defer func() {
		if r := recover(); r != nil {
			logx.WithContext(ctx).Errorf("rpc panic: %s: %v\n%s", unaryMethodName(info), r, debug.Stack())
			resp, err = nil, errx.SystemBusyError()
		}
		err = errx.ToGRPCStatus(err)
	}()
	return handler(ctx, req)
}

// ServerStreamFinalizeInterceptor 是服务端流式 RPC 的最外层拦截器，必须注册在
// go-zero 内置 stream 拦截器之外，职责与一元一致：兜底 panic、归一化出口错误。
func ServerStreamFinalizeInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {
	ctx := ExtractFromGrpcMD(ss.Context())
	defer func() {
		if r := recover(); r != nil {
			logx.WithContext(ctx).Errorf("rpc panic: %s: %v\n%s", streamMethodName(info), r, debug.Stack())
			err = errx.SystemBusyError()
		}
		err = errx.ToGRPCStatus(err)
	}()
	return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
}

func unaryMethodName(info *grpc.UnaryServerInfo) string {
	if info == nil {
		return ""
	}
	return info.FullMethod
}

func streamMethodName(info *grpc.StreamServerInfo) string {
	if info == nil {
		return ""
	}
	return info.FullMethod
}
