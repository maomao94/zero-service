package grpcx

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryMetadataInterceptor injects propagatable context values into outgoing
// metadata before invoking a unary RPC.
func UnaryMetadataInterceptor(ctx context.Context, method string, req, reply any,
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(InjectToGrpcMD(ctx), method, req, reply, cc, opts...)
}

// StreamTracingInterceptor injects propagatable context values into outgoing
// metadata before opening a streaming RPC.
func StreamTracingInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
	method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(InjectToGrpcMD(ctx), desc, cc, method, opts...)
}
