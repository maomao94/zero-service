package grpcx

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/authctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryMetadataInterceptor(t *testing.T) {
	wantErr := errors.New("invoke failed")
	wantReq := &struct{}{}
	wantReply := &struct{}{}
	wantMethod := "/test.Service/Unary"
	wantOption := grpc.WaitForReady(true)
	ctx := authctx.WithUserID(context.Background(), "user-1")
	called := false

	err := UnaryMetadataInterceptor(ctx, wantMethod, wantReq, wantReply, nil,
		func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			called = true
			if method != wantMethod || req != wantReq || reply != wantReply || cc != nil {
				t.Fatalf("invoker arguments were not passed through")
			}
			if len(opts) != 1 || opts[0] != wantOption {
				t.Fatalf("invoker options = %v, want configured option", opts)
			}
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok || len(md.Get(HeaderUserId)) != 1 || md.Get(HeaderUserId)[0] != "user-1" {
				t.Fatalf("outgoing metadata = %v, want propagated user id", md)
			}
			return wantErr
		}, wantOption)

	if !called {
		t.Fatal("invoker was not called")
	}
	if err != wantErr {
		t.Fatalf("error = %v, want identical error %v", err, wantErr)
	}
}

func TestStreamTracingInterceptor(t *testing.T) {
	wantErr := errors.New("stream failed")
	wantStream := &clientStreamStub{ctx: context.Background()}
	wantDesc := &grpc.StreamDesc{StreamName: "Watch"}
	wantMethod := "/test.Service/Watch"
	wantOption := grpc.WaitForReady(true)
	ctx := authctx.WithDeptCode(context.Background(), "dept-1")
	called := false

	gotStream, err := StreamTracingInterceptor(ctx, wantDesc, nil, wantMethod,
		func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			called = true
			if desc != wantDesc || cc != nil || method != wantMethod {
				t.Fatalf("streamer arguments were not passed through")
			}
			if len(opts) != 1 || opts[0] != wantOption {
				t.Fatalf("streamer options = %v, want configured option", opts)
			}
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok || len(md.Get(HeaderDeptCode)) != 1 || md.Get(HeaderDeptCode)[0] != "dept-1" {
				t.Fatalf("outgoing metadata = %v, want propagated department code", md)
			}
			return wantStream, wantErr
		}, wantOption)

	if !called {
		t.Fatal("streamer was not called")
	}
	if gotStream != wantStream {
		t.Fatalf("stream = %p, want %p", gotStream, wantStream)
	}
	if err != wantErr {
		t.Fatalf("error = %v, want identical error %v", err, wantErr)
	}
}

type clientStreamStub struct {
	grpc.ClientStream
	ctx context.Context
}

func (s *clientStreamStub) Context() context.Context {
	return s.ctx
}
