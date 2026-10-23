package errx

import (
	"testing"

	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCodeFromStatus(t *testing.T) {
	withReason, err := status.New(codes.Unavailable, "downstream down").
		WithDetails(&errdetails.ErrorInfo{Reason: "100601"})
	if err != nil {
		t.Fatalf("with details: %v", err)
	}
	withDJIReason, err := status.New(codes.FailedPrecondition, "设备升级中").
		WithDetails(&errdetails.ErrorInfo{Reason: "312014", Domain: "dji"})
	if err != nil {
		t.Fatalf("with dji details: %v", err)
	}

	tests := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{name: "nil", err: nil, code: 0, msg: ""},
		{name: "error info reason", err: withReason.Err(), code: 100601, msg: "downstream down"},
		{name: "dji reason rejected", err: withDJIReason.Err(), code: 100998, msg: ""},
		{name: "not found fallback", err: status.Error(codes.NotFound, "missing"), code: 100202, msg: "missing"},
		{name: "deadline fallback", err: status.Error(codes.DeadlineExceeded, ""), code: 100997, msg: "系统超时"},
		{name: "code msg", err: xerrors.New(130102, "bad secret"), code: 130102, msg: "bad secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := CodeFromStatus(tt.err)
			if code != tt.code {
				t.Fatalf("code = %d, want %d", code, tt.code)
			}
			if msg != tt.msg {
				t.Fatalf("msg = %q, want %q", msg, tt.msg)
			}
		})
	}
}

// extproto 是平台码表：只允许 A=0（成功）与 A=1（平台服务端）。
// 设备/云端原始码必须留在厂商自己的枚举里，不得占用平台命名空间。
func TestExtprotoStaysInPlatformNamespace(t *testing.T) {
	for num := range extproto.Code_name {
		if num == 0 {
			continue // 0 是唯一成功码
		}
		switch SourceOf(int(num)) {
		case 1:
		default:
			t.Fatalf("错误码 %d 的来源位 A=%d，extproto 只允许 0/1", num, SourceOf(int(num)))
		}
	}
}

// 未在 extproto 中定义的码必须降级为 100999。
func TestNormalizeErrorRejectsUndefinedCode(t *testing.T) {
	code, msg := NormalizeError(xerrors.New(100555, "made up"))
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != SystemBusyMessage {
		t.Fatalf("NormalizeError() = %d/%q, want 100999/%q", code, msg, SystemBusyMessage)
	}
}

// 已定义的码必须原样保留。
func TestNormalizeErrorKeepsDefinedCode(t *testing.T) {
	code, msg := NormalizeError(xerrors.New(int(extproto.Code__1_30_APP_NOT_FOUND), "应用不存在或已停用"))
	if code != int(extproto.Code__1_30_APP_NOT_FOUND) || msg != "应用不存在或已停用" {
		t.Fatalf("NormalizeError() = %d/%q", code, msg)
	}
}

func TestFoldPublic(t *testing.T) {
	tests := map[int]int{
		0:      0,
		100998: int(extproto.Code__1_00_UNKNOWN),
		100201: int(extproto.Code__1_00_UNKNOWN),
		100301: int(extproto.Code__1_00_UNKNOWN),
		100601: int(extproto.Code__1_00_UNKNOWN),
		100101: 100101,
		130101: 130101,
	}
	for in, want := range tests {
		if got := FoldPublic(in); got != want {
			t.Fatalf("FoldPublic(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestPublicMessage(t *testing.T) {
	if got := PublicMessage(100601, "db down"); got != "系统繁忙" {
		t.Fatalf("folded message = %q", got)
	}
	if got := PublicMessage(130101, "应用不存在或已停用"); got != "应用不存在或已停用" {
		t.Fatalf("public message = %q", got)
	}
}

func TestSuccessCodeIsZero(t *testing.T) {
	if SuccessCode != 0 {
		t.Fatalf("success code must be 0, got %d", SuccessCode)
	}
	if !IsSuccess(0) || IsSuccess(1) {
		t.Fatal("IsSuccess must only accept 0")
	}
	if int(extproto.Code__0_OK) != 0 {
		t.Fatal("extproto success code must be 0")
	}
}
