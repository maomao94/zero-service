package tool

import (
	"testing"

	"zero-service/third_party/extproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewErrorByPbCodeFormatsCustomMessage(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "第 %d 个参数错误: %s", 2, "value")

	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("status.Code() = %v, want %v", got, codes.InvalidArgument)
	}
	if got := status.Convert(err).Message(); got != "第 2 个参数错误: value" {
		t.Fatalf("status message = %q, want formatted message", got)
	}
}

func TestNewErrorByPbCodeUsesPlainCustomMessage(t *testing.T) {
	err := NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "参数错误")

	if got := status.Convert(err).Message(); got != "参数错误" {
		t.Fatalf("status message = %q, want plain custom message", got)
	}
}
