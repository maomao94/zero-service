package grpcx

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type appIDRequest struct {
	AppID string
}

func (r *appIDRequest) GetAppId() string { return r.AppID }

type plainRequest struct{}

func TestAppIDMetadataInterceptor(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("existing", "1"))
	called := false

	err := AppIDMetadataInterceptor(ctx, "/openapi.OpenApiRpc/Invoke", &appIDRequest{AppID: "app_123"}, nil, nil,
		func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			called = true
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("outgoing metadata missing")
			}
			if got := md.Get(HeaderAppID); len(got) != 1 || got[0] != "app_123" {
				t.Fatalf("app id metadata = %#v", got)
			}
			if got := md.Get("existing"); len(got) != 1 || got[0] != "1" {
				t.Fatalf("existing metadata was not preserved: %#v", got)
			}
			return nil
		})

	if !called {
		t.Fatal("invoker was not called")
	}
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
}

func TestAppIDMetadataInterceptorSkipsEmptyOrUnknownRequest(t *testing.T) {
	for _, req := range []any{&appIDRequest{}, &plainRequest{}} {
		called := false
		err := AppIDMetadataInterceptor(context.Background(), "/test.Service/Call", req, nil, nil,
			func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				called = true
				md, _ := metadata.FromOutgoingContext(ctx)
				if got := md.Get(HeaderAppID); len(got) != 0 {
					t.Fatalf("app id metadata should be empty, got %#v", got)
				}
				return nil
			})
		if err != nil || !called {
			t.Fatalf("req %T: err=%v called=%v", req, err, called)
		}
	}
}
