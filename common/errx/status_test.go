package errx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCStatusNil(t *testing.T) {
	if got := ToGRPCStatus(nil); got != nil {
		t.Fatalf("ToGRPCStatus(nil) = %v, want nil", got)
	}
}

func TestToGRPCStatusKeepsContextSemantics(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   codes.Code
		reason extproto.Code
	}{
		{name: "canceled", err: context.Canceled, want: codes.Canceled, reason: extproto.Code__1_00_CANCELED},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: codes.DeadlineExceeded, reason: extproto.Code__1_00_TIMEOUT},
		{name: "wrapped deadline exceeded", err: fmt.Errorf("call downstream: %w", context.DeadlineExceeded), want: codes.DeadlineExceeded, reason: extproto.Code__1_00_TIMEOUT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToGRPCStatus(tt.err)
			if code := status.Code(got); code != tt.want {
				t.Fatalf("code = %v, want %v", code, tt.want)
			}
			if msg := status.Convert(got).Message(); msg != tt.err.Error() {
				t.Fatalf("message = %q, want %q", msg, tt.err.Error())
			}
			assertReason(t, got, int(tt.reason))
		})
	}
}

func TestToGRPCStatusKeepsPlatformCodeWrappingContext(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "wrapped deadline", err: NewErrorByPbCodeWrap(extproto.Code__1_00_DB, context.DeadlineExceeded, "查询失败")},
		{name: "wrapped canceled", err: NewErrorByPbCodeWrap(extproto.Code__1_00_DB, context.Canceled, "查询失败")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToGRPCStatus(tt.err)
			assertStatus(t, got, codes.Internal, "查询失败")
			assertReason(t, got, int(extproto.Code__1_00_DB))
		})
	}
}

func TestToGRPCStatusSanitizesPlainError(t *testing.T) {
	cause := errors.New(`dial tcp 10.0.0.1:5432: password authentication failed for user "zero"`)

	got := ToGRPCStatus(cause)

	assertStatus(t, got, codes.Internal, SystemBusyMessage)
	assertReason(t, got, int(extproto.Code__1_00_UNKNOWN))
	if msg := status.Convert(got).Message(); strings.Contains(msg, "10.0.0.1") || strings.Contains(msg, "password") {
		t.Fatalf("status message leaks internal detail: %q", msg)
	}
}

func TestToGRPCStatusAddsFallbackDetailForExplicitStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code extproto.Code
	}{
		{name: "not found", err: status.Error(codes.NotFound, "设备不存在"), code: extproto.Code__1_00_RECORD_NOT_EXIST},
		{name: "unknown", err: status.Error(codes.Unknown, "未知错误"), code: extproto.Code__1_00_INTERNAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToGRPCStatus(tt.err)
			assertStatus(t, got, status.Code(tt.err), status.Convert(tt.err).Message())
			assertReason(t, got, int(tt.code))
		})
	}
}

func TestToGRPCStatusConvertsCodeMsg(t *testing.T) {
	got := ToGRPCStatus(xerrors.New(int(extproto.Code__1_00_DB), "查询配置失败"))

	assertStatus(t, got, codes.Internal, "查询配置失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
}

func TestToGRPCStatusRejectsInvalidCodeMsg(t *testing.T) {
	got := ToGRPCStatus(xerrors.New(100555, "内部细节"))

	assertStatus(t, got, codes.Internal, SystemBusyMessage)
	assertReason(t, got, int(extproto.Code__1_00_UNKNOWN))
}

// 所有标准 gRPC Code 都必须能补出 extproto detail，且 detail 的 http_code
// 与该 gRPC Code 的语义 HTTP 一致。
func TestToGRPCStatusAlwaysAddsDetail(t *testing.T) {
	for code := codes.OK; code <= codes.Unauthenticated; code++ {
		if code == codes.OK {
			continue
		}
		t.Run(code.String(), func(t *testing.T) {
			got := ToGRPCStatus(status.Error(code, "x"))
			extprotoCode, ok := platformCodeOf(got)
			if !ok {
				t.Fatalf("status has no extproto detail: %v", got)
			}
			if gotHTTP, wantHTTP := semanticHTTPCode(extprotoCode), GrpcCodeToHTTPStatus(code); gotHTTP != wantHTTP {
				t.Fatalf("detail code %d http_code=%d, gRPC code semantic HTTP=%d", extprotoCode, gotHTTP, wantHTTP)
			}
		})
	}
}

// 每个 extproto 码的 http_code 都必须登记在正向映射表中，且反向分类回同一个
// http_code，保证业务码、gRPC Code、语义 HTTP 三者一致。
func TestExtprotoHTTPCodeGRPCCodeRoundTrip(t *testing.T) {
	for num := range extproto.Code_name {
		httpCode := semanticHTTPCode(int(num))
		grpcCode, ok := httpCodeToGRPCCode[httpCode]
		if !ok {
			t.Fatalf("code=%d http_code=%d 未登记 gRPC Code 映射", num, httpCode)
		}
		if got := GrpcCodeToHTTPStatus(grpcCode); got != httpCode {
			t.Fatalf("code=%d http_code=%d grpc=%v 反向得到 %d，want %d",
				num, httpCode, grpcCode, got, httpCode)
		}
	}
}

// 每个 extproto 码都必须满足：gRPC Code 是标准 gRPC code（0-16），
// 六位业务码只出现在 ErrorInfo.reason，不会被写进 gRPC Code。
func TestNewErrorByPbCodeKeepsGRPCCodeStandardAndBusinessCodeInDetail(t *testing.T) {
	for num := range extproto.Code_name {
		if num == 0 {
			continue
		}
		code := extproto.Code(num)
		err := NewErrorByPbCode(code)

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("code=%d error is not a gRPC status", num)
		}
		if int(st.Code()) >= 100 {
			t.Fatalf("code=%d gRPC code=%d，业务码不能写入 gRPC Code", num, st.Code())
		}
		if want := grpcCodeFromHTTPStatus(semanticHTTPCode(int(num))); st.Code() != want {
			t.Fatalf("code=%d gRPC code=%v, want %v", num, st.Code(), want)
		}
		if got, _ := CodeFromStatus(err); got != int(num) {
			t.Fatalf("code=%d detail 解析为 %d", num, got)
		}
	}
}

func TestToGRPCStatusDoesNotOverrideExistingDetail(t *testing.T) {
	original, err := status.New(codes.InvalidArgument, "厂商拒绝").
		WithDetails(&errdetails.ErrorInfo{Reason: "312014", Domain: "dji"})
	if err != nil {
		t.Fatalf("with details: %v", err)
	}

	got := ToGRPCStatus(original.Err())

	if reasons := rpcReasons(t, got); len(reasons) != 1 || reasons[0] != "312014" {
		t.Fatalf("reasons = %v, want only vendor detail preserved", reasons)
	}
}

func rpcReasons(t *testing.T, err error) []string {
	t.Helper()

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	var reasons []string
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		reasons = append(reasons, info.GetReason())
	}
	return reasons
}

func TestToGRPCStatusKeepsBusinessCode(t *testing.T) {
	got := ToGRPCStatus(NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败"))

	assertStatus(t, got, codes.Internal, "查询设备失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
}

func TestToGRPCStatusKeepsWrappedBusinessCode(t *testing.T) {
	cause := errors.New(`pq: password authentication failed for user "zero"`)
	got := ToGRPCStatus(NewErrorByPbCodeWrap(extproto.Code__1_00_DB, cause, "查询设备失败"))

	assertStatus(t, got, codes.Internal, "查询设备失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
	if strings.Contains(got.Error(), "password") {
		t.Fatalf("error leaks cause: %v", got)
	}
}

func TestToGRPCStatusHidesFmtWrapStatus(t *testing.T) {
	inner := status.Error(codes.NotFound, "设备不存在")
	got := ToGRPCStatus(fmt.Errorf("logic failed: %w", inner))

	assertStatus(t, got, codes.NotFound, "设备不存在")
	assertReason(t, got, int(extproto.Code__1_00_RECORD_NOT_EXIST))
}

func TestToGRPCStatusHidesFmtWrapCause(t *testing.T) {
	businessErr := NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")
	got := ToGRPCStatus(fmt.Errorf("logic failed: %w", businessErr))

	assertStatus(t, got, codes.Internal, "查询设备失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
	if strings.Contains(got.Error(), "logic failed") {
		t.Fatalf("error leaks fmt wrap message: %v", got)
	}
}

func TestToGRPCStatusKeepsExplicitStatus(t *testing.T) {
	got := ToGRPCStatus(status.Error(codes.NotFound, "设备不存在"))

	assertStatus(t, got, codes.NotFound, "设备不存在")
}

func assertStatus(t *testing.T, err error, wantCode codes.Code, wantMsg string) {
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

func assertReason(t *testing.T, err error, want int) {
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
		if got := info.GetReason(); got != fmt.Sprintf("%d", want) {
			t.Fatalf("reason = %q, want %d", got, want)
		}
		return
	}
	t.Fatalf("error %v has no ErrorInfo detail", err)
}
