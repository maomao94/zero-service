package grpcx

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"zero-service/common/authctx"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_UNKNOWN) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_UNKNOWN))
	}
}

func TestLoggerInterceptorKeepsBusinessCode(t *testing.T) {
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")

	_, err := LoggerInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			return nil, wantErr
		})

	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

func TestLoggerInterceptorKeepsContextError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		code   codes.Code
		reason extproto.Code
	}{
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded, reason: extproto.Code__1_00_TIMEOUT},
		{name: "canceled", err: context.Canceled, code: codes.Canceled, reason: extproto.Code__1_00_CANCELED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoggerInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
				func(ctx context.Context, req any) (any, error) {
					return nil, tt.err
				})

			if code := status.Code(err); code != tt.code {
				t.Fatalf("code = %v, want %v", code, tt.code)
			}
			if reason := rpcErrorReason(t, err); reason != int(tt.reason) {
				t.Fatalf("reason = %d, want %d", reason, int(tt.reason))
			}
		})
	}
}

func TestLoggerInterceptorRecoversPanic(t *testing.T) {
	gotResp, err := LoggerInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			panic("boom: password=secret")
		})

	if gotResp != nil {
		t.Fatalf("response = %v, want nil", gotResp)
	}
	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_UNKNOWN) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_UNKNOWN))
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
	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
}

func TestStreamLoggerInterceptorKeepsBusinessCode(t *testing.T) {
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")
	originalStream := &serverStreamStub{ctx: context.Background()}

	err := StreamLoggerInterceptor(&struct{}{}, originalStream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			return wantErr
		})

	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

func TestStreamLoggerInterceptorRecoversPanic(t *testing.T) {
	originalStream := &serverStreamStub{ctx: context.Background()}

	err := StreamLoggerInterceptor(&struct{}{}, originalStream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			panic("boom: password=secret")
		})

	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_UNKNOWN) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_UNKNOWN))
	}
}

func TestServerFinalizeInterceptorKeepsBusinessCode(t *testing.T) {
	wantResp := &struct{}{}
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")

	gotResp, err := ServerFinalizeInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			return wantResp, wantErr
		})

	if gotResp != wantResp {
		t.Fatalf("response = %v, want %v", gotResp, wantResp)
	}
	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

func TestServerFinalizeInterceptorNormalizesBareError(t *testing.T) {
	_, err := ServerFinalizeInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			return nil, errors.New("db down")
		})

	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_UNKNOWN) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_UNKNOWN))
	}
}

func TestServerFinalizeInterceptorRecoversPanic(t *testing.T) {
	gotResp, err := ServerFinalizeInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			panic("boom: password=secret")
		})

	if gotResp != nil {
		t.Fatalf("response = %v, want nil", gotResp)
	}
	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
}

func TestServerFinalizeInterceptorWithLoggerInterceptor(t *testing.T) {
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")

	_, err := ServerFinalizeInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
		func(ctx context.Context, req any) (any, error) {
			return LoggerInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Unary"},
				func(ctx context.Context, req any) (any, error) {
					return nil, wantErr
				})
		})

	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

func TestServerStreamFinalizeInterceptorKeepsBusinessCode(t *testing.T) {
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")
	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(HeaderUserId, "user-2"))
	stream := &serverStreamStub{ctx: baseCtx}

	err := ServerStreamFinalizeInterceptor(&struct{}{}, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			if got := authctx.GetUserId(stream.Context()); got != "user-2" {
				t.Fatalf("user id = %q, want %q", got, "user-2")
			}
			return wantErr
		})

	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

func TestServerStreamFinalizeInterceptorRecoversPanic(t *testing.T) {
	stream := &serverStreamStub{ctx: context.Background()}

	err := ServerStreamFinalizeInterceptor(&struct{}{}, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			panic("boom: password=secret")
		})

	assertRPCStatus(t, err, codes.Internal, errx.SystemBusyMessage)
}

func TestServerStreamFinalizeInterceptorWithLoggerInterceptor(t *testing.T) {
	wantErr := errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")
	stream := &serverStreamStub{ctx: context.Background()}

	err := ServerStreamFinalizeInterceptor(&struct{}{}, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
		func(srv any, stream grpc.ServerStream) error {
			return StreamLoggerInterceptor(srv, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/Watch"},
				func(srv any, stream grpc.ServerStream) error {
					return wantErr
				})
		})

	assertRPCStatus(t, err, codes.Internal, "查询设备失败")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_DB) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_DB))
	}
}

// TestServerFinalizeInterceptorAddsDefaultDetailForTimeout 验证超时这类没有
// ErrorInfo 的 status 会被补上默认 detail，客户端仍能按六位码处理。
func TestServerFinalizeInterceptorAddsDefaultDetailForTimeout(t *testing.T) {
	timeoutErr := status.Error(codes.DeadlineExceeded, "context deadline exceeded")

	_, err := ServerFinalizeInterceptor(context.Background(), &struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Slow"},
		func(ctx context.Context, req any) (any, error) {
			return nil, timeoutErr
		})

	assertRPCStatus(t, err, codes.DeadlineExceeded, "context deadline exceeded")
	if reason := rpcErrorReason(t, err); reason != int(extproto.Code__1_00_TIMEOUT) {
		t.Fatalf("reason = %d, want %d", reason, int(extproto.Code__1_00_TIMEOUT))
	}
}

type serverStreamStub struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamStub) Context() context.Context {
	return s.ctx
}

func assertRPCStatus(t *testing.T, err error, wantCode codes.Code, wantMsg string) {
	t.Helper()

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	if st.Code() != wantCode {
		t.Fatalf("code = %v, want %v", st.Code(), wantCode)
	}
	if st.Message() != wantMsg {
		t.Fatalf("message = %q, want %q", st.Message(), wantMsg)
	}
}

func rpcErrorReason(t *testing.T, err error) int {
	t.Helper()

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		reason, convErr := strconv.Atoi(info.GetReason())
		if convErr != nil {
			t.Fatalf("reason %q is not numeric: %v", info.GetReason(), convErr)
		}
		return reason
	}
	t.Fatalf("error %v has no ErrorInfo detail", err)
	return 0
}
