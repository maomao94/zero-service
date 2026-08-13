package grpcx

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/authctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestLoggerInterceptor(t *testing.T) {
	wantErr := errors.New("handler failed")
	wantReq := &struct{}{}
	wantResp := &struct{}{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(HeaderUserName, "alice"))
	called := false

	gotResp, err := LoggerInterceptor(ctx, wantReq, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			called = true
			if req != wantReq {
				t.Fatalf("request = %v, want original request", req)
			}
			if got := authctx.GetUserName(ctx); got != "alice" {
				t.Fatalf("user name = %q, want %q", got, "alice")
			}
			return wantResp, wantErr
		})

	if !called {
		t.Fatal("handler was not called")
	}
	if gotResp != wantResp {
		t.Fatalf("response = %v, want original response", gotResp)
	}
	if err != wantErr {
		t.Fatalf("error = %v, want identical error %v", err, wantErr)
	}
}

func TestStreamLoggerInterceptor(t *testing.T) {
	wantErr := errors.New("stream handler failed")
	wantSrv := &struct{}{}
	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(HeaderUserId, "user-2"))
	originalStream := &serverStreamStub{ctx: baseCtx}
	called := false

	err := StreamLoggerInterceptor(wantSrv, originalStream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			called = true
			if srv != wantSrv {
				t.Fatalf("service = %v, want original service", srv)
			}
			if stream == originalStream {
				t.Fatal("handler received original stream, want context-wrapping stream")
			}
			if got := authctx.GetUserId(stream.Context()); got != "user-2" {
				t.Fatalf("user id = %q, want %q", got, "user-2")
			}
			return wantErr
		})

	if !called {
		t.Fatal("handler was not called")
	}
	if err != wantErr {
		t.Fatalf("error = %v, want identical error %v", err, wantErr)
	}
}

type serverStreamStub struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamStub) Context() context.Context {
	return s.ctx
}
