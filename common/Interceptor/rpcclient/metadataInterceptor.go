package interceptor

import (
	"context"
	"zero-service/common/ctxdata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryMetadataInterceptor(ctx context.Context, method string, req, reply any,
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	if userId := ctxdata.GetUserId(ctx); userId != "" {
		md.Set(ctxdata.HeaderUserId, userId)
	}
	if userName := ctxdata.GetUserName(ctx); userName != "" {
		md.Set(ctxdata.HeaderUserName, userName)
	}
	if deptCode := ctxdata.GetDeptCode(ctx); deptCode != "" {
		md.Set(ctxdata.HeaderDeptCode, deptCode)
	}
	if auth := ctxdata.GetAuthorization(ctx); auth != "" {
		md.Set(ctxdata.HeaderAuthorization, auth)
	}
	if traceId := ctxdata.GetTraceId(ctx); traceId != "" {
		md.Set(ctxdata.HeaderTraceId, traceId)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	return invoker(ctx, method, req, reply, cc, opts...)
}

func StreamTracingInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
	method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	if userId := ctxdata.GetUserId(ctx); userId != "" {
		md.Set(ctxdata.HeaderUserId, userId)
	}
	if userName := ctxdata.GetUserName(ctx); userName != "" {
		md.Set(ctxdata.HeaderUserName, userName)
	}
	if deptCode := ctxdata.GetDeptCode(ctx); deptCode != "" {
		md.Set(ctxdata.HeaderDeptCode, deptCode)
	}
	if auth := ctxdata.GetAuthorization(ctx); auth != "" {
		md.Set(ctxdata.HeaderAuthorization, auth)
	}
	if traceId := ctxdata.GetTraceId(ctx); traceId != "" {
		md.Set(ctxdata.HeaderTraceId, traceId)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	return streamer(ctx, desc, cc, method, opts...)
}
