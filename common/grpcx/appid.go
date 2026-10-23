package grpcx

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// HeaderAppID 是开放平台应用 ID 的 outgoing metadata key。
const HeaderAppID = "x-app-id"

type appIDGetter interface {
	GetAppId() string
}

// AppIDMetadataInterceptor 将请求中的 app_id 追加到 outgoing metadata。
// 仅对实现 GetAppId() 的请求生效，供开放平台等需要透传应用身份的调用方使用。
func AppIDMetadataInterceptor(ctx context.Context, method string, req, reply any,
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if getter, ok := req.(appIDGetter); ok {
		if appID := getter.GetAppId(); appID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, HeaderAppID, appID)
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
