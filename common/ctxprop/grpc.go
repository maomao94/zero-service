package ctxprop

import (
	"context"

	"zero-service/common/ctxdata"

	"google.golang.org/grpc/metadata"
)

// InjectToGrpcMD 从 context values 提取所有字段，注入到 outgoing gRPC metadata。
// 用于 gRPC 客户端拦截器：将上下文字段传播到下游 RPC 服务。
func InjectToGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	for _, f := range ctxdata.PropFields {
		if v, ok := ctx.Value(f.CtxKey).(string); ok && v != "" {
			md.Set(f.GrpcHeader, v)
		}
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractFromGrpcMD 从 incoming gRPC metadata 提取所有字段，注入到 context values。
// 用于 gRPC 服务端拦截器：将 metadata 中的字段恢复到 context 供业务层使用。
func ExtractFromGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, f := range ctxdata.PropFields {
		if v := md.Get(f.GrpcHeader); len(v) > 0 {
			ctx = context.WithValue(ctx, f.CtxKey, v[0])
		}
	}
	return ctx
}
