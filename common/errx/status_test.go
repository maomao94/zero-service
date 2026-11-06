package errx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		name string
		err  error
		want codes.Code
	}{
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
		{name: "wrapped deadline exceeded", err: fmt.Errorf("call downstream: %w", context.DeadlineExceeded), want: codes.DeadlineExceeded},
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
		})
	}
}

func TestToGRPCStatusSanitizesPlainError(t *testing.T) {
	cause := errors.New(`dial tcp 10.0.0.1:5432: password authentication failed for user "zero"`)

	got := ToGRPCStatus(cause)

	assertStatus(t, got, codes.Internal, SystemBusyMessage)
	if strings.Contains(got.Error(), "10.0.0.1") || strings.Contains(got.Error(), "password") {
		t.Fatalf("error leaks internal detail: %v", got)
	}
}

func TestToGRPCStatusKeepsBusinessCode(t *testing.T) {
	got := ToGRPCStatus(tool.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败"))

	assertStatus(t, got, codes.Internal, "查询设备失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
}

func TestToGRPCStatusKeepsWrappedBusinessCode(t *testing.T) {
	cause := errors.New(`pq: password authentication failed for user "zero"`)
	got := ToGRPCStatus(tool.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, cause, "查询设备失败"))

	assertStatus(t, got, codes.Internal, "查询设备失败")
	assertReason(t, got, int(extproto.Code__1_00_DB))
	if strings.Contains(got.Error(), "password") {
		t.Fatalf("error leaks cause: %v", got)
	}
}

func TestToGRPCStatusHidesFmtWrapCause(t *testing.T) {
	businessErr := tool.NewErrorByPbCode(extproto.Code__1_00_DB, "查询设备失败")
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
