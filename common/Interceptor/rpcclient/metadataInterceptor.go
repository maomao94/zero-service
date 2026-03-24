package interceptor

import (
	"context"

	"zero-service/common/ctxprop"

	"google.golang.org/grpc"
)

func UnaryMetadataInterceptor(ctx context.Context, method string, req, reply any,
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(ctxprop.InjectToGrpcMD(ctx), method, req, reply, cc, opts...)
}

func StreamTracingInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
	method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(ctxprop.InjectToGrpcMD(ctx), desc, cc, method, opts...)
}
