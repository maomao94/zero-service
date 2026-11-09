package errx

import (
	"errors"
	"fmt"
	"testing"

	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewErrorByPbCodeFormatsCustomMessage(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "第 %d 个参数错误: %s", 2, "value")

	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("status.Code() = %v, want %v", got, codes.InvalidArgument)
	}
	if got := status.Convert(err).Message(); got != "第 2 个参数错误: value" {
		t.Fatalf("status message = %q, want formatted message", got)
	}
}

func TestNewErrorByPbCodeUsesPlainCustomMessage(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "参数错误")

	if got := status.Convert(err).Message(); got != "参数错误" {
		t.Fatalf("status message = %q, want plain custom message", got)
	}
}

func TestNewErrorByPbCodeRejectsSuccessCode(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__0_OK)

	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("status.Code() = %v, want %v", got, codes.Internal)
	}
	if got := status.Convert(err).Message(); got == "" || got == "成功" {
		t.Fatalf("status message = %q, want an error message", got)
	}
	if !IsErrorByPbCode(err, extproto.Code__1_00_INTERNAL) {
		t.Fatalf("success code should degrade to %d", extproto.Code__1_00_INTERNAL)
	}
}

func TestNewErrorByPbCodeRejectsInvalidCode(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code(100555), "非法码")

	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("status.Code() = %v, want %v", got, codes.Internal)
	}
	code, msg := CodeFromStatus(err)
	if code != int(extproto.Code__1_00_INTERNAL) || msg != "非法错误码不能用于构造错误" {
		t.Fatalf("CodeFromStatus() = %d/%q, want %d/%q",
			code, msg, extproto.Code__1_00_INTERNAL, "非法错误码不能用于构造错误")
	}
}

func TestNewErrorByPbCodeRateLimited(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__1_00_RATE_LIMITED)

	if got := status.Code(err); got != codes.ResourceExhausted {
		t.Fatalf("status.Code() = %v, want %v", got, codes.ResourceExhausted)
	}
	if !IsErrorByPbCode(err, extproto.Code__1_00_RATE_LIMITED) {
		t.Fatalf("reason = %v, want 100220", err)
	}
}

func TestNewErrorByPbCodeWrapKeepsCodeAndHidesCause(t *testing.T) {
	cause := errors.New(`pq: password authentication failed for user "zero"`)
	err := NewErrorByPbCodeWrap(extproto.Code__1_00_DB, cause, "查询设备失败")

	if !IsErrorByPbCode(err, extproto.Code__1_00_DB) {
		t.Fatalf("code = %d, want %d", status.Code(err), extproto.Code__1_00_DB)
	}
	if got := status.Convert(err).Message(); got != "查询设备失败" {
		t.Fatalf("status message = %q, want %q", got, "查询设备失败")
	}
	if errors.Is(err, cause) == false {
		t.Fatal("wrapped error should keep cause for errors.Is")
	}
	if got := status.Convert(err).Message(); got == cause.Error() {
		t.Fatal("cause must not leak into gRPC status message")
	}
}

func TestNewErrorByPbCodeWrapNilCause(t *testing.T) {
	err := NewErrorByPbCodeWrap(extproto.Code__1_00_DB, nil, "查询设备失败")

	if !IsErrorByPbCode(err, extproto.Code__1_00_DB) {
		t.Fatalf("code = %v, want %d", status.Code(err), extproto.Code__1_00_DB)
	}
	if got := err.Error(); got != "查询设备失败" {
		t.Fatalf("Error() = %q, want %q", got, "查询设备失败")
	}
}

func TestNewErrorByPbCodeWrapPreservesExistingPlatformCode(t *testing.T) {
	tests := []struct {
		name    string
		cause   error
		code    extproto.Code
		message string
	}{
		{
			name:    "errx error",
			cause:   NewErrorByPbCode(extproto.Code__1_00_RECORD_NOT_EXIST, "Modbus配置不存在: dev-1"),
			code:    extproto.Code__1_00_RECORD_NOT_EXIST,
			message: "Modbus配置不存在: dev-1",
		},
		{
			name:    "code msg",
			cause:   xerrors.New(int(extproto.Code__1_00_DB), "查询配置失败"),
			code:    extproto.Code__1_00_DB,
			message: "查询配置失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewErrorByPbCodeWrap(extproto.Code__1_00_BIZ, tt.cause, "获取 Modbus 客户端失败")
			code, msg := CodeFromStatus(got)
			if code != int(tt.code) || msg != tt.message {
				t.Fatalf("CodeFromStatus() = %d/%q, want %d/%q", code, msg, tt.code, tt.message)
			}
		})
	}
}

func TestNewErrorByPbCodeWrapConvertsCodeMsgToStatus(t *testing.T) {
	got := NewErrorByPbCodeWrap(
		extproto.Code__1_00_BIZ,
		xerrors.New(int(extproto.Code__1_00_DB), "查询配置失败"),
		"外层上下文",
	)

	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("wrapped CodeMsg is not a gRPC status: %v", got)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("status.Code() = %v, want %v", st.Code(), codes.Internal)
	}
	if code, msg := CodeFromStatus(got); code != int(extproto.Code__1_00_DB) || msg != "查询配置失败" {
		t.Fatalf("CodeFromStatus() = %d/%q, want %d/%q",
			code, msg, extproto.Code__1_00_DB, "查询配置失败")
	}
}

func TestPublicErrorMatchesCodeFromStatus(t *testing.T) {
	got := NewErrorByPbCodeWrap(
		extproto.Code__1_00_DB,
		xerrors.New(100555, "非法内部码"),
		"查询失败",
	)

	codeFromStatus, msgFromStatus := CodeFromStatus(got)
	publicCode, publicMsg := PublicError(got)
	if codeFromStatus != publicCode || msgFromStatus != publicMsg {
		t.Fatalf("CodeFromStatus() = %d/%q, PublicError() = %d/%q",
			codeFromStatus, msgFromStatus, publicCode, publicMsg)
	}
	if publicCode != int(extproto.Code__1_00_DB) || publicMsg != "查询失败" {
		t.Fatalf("PublicError() = %d/%q, want %d/%q",
			publicCode, publicMsg, extproto.Code__1_00_DB, "查询失败")
	}
}

func TestIsErrorByPbCodeSupportsStatusAndCodeMsg(t *testing.T) {
	grpcErr := status.Error(codes.NotFound, "记录不存在")
	if !IsErrorByPbCode(grpcErr, extproto.Code__1_00_RECORD_NOT_EXIST) {
		t.Fatal("grpc status should fall back to mapped extproto code")
	}

	codeMsgErr := xerrors.New(int(extproto.Code__1_30_APP_SECRET_INVALID), "密钥无效")
	if !IsErrorByPbCode(fmt.Errorf("auth: %w", codeMsgErr), extproto.Code__1_30_APP_SECRET_INVALID) {
		t.Fatal("wrapped CodeMsg should keep its extproto code")
	}

	if IsErrorByPbCode(errors.New("boom"), extproto.Code__1_00_UNKNOWN) {
		t.Fatal("plain error should not match an extproto code")
	}
}
