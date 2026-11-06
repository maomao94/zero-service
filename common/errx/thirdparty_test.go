package errx

import (
	"fmt"
	"strings"
	"testing"

	"zero-service/third_party/extproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestThirdPartyAPIErrorTemplate(t *testing.T) {
	if n := strings.Count(ThirdPartyAPIErrorTemplate, "%s"); n != 3 {
		t.Fatalf("template placeholders = %d, want 3", n)
	}
	if n := strings.Count(ThirdPartyAPIErrorTemplateNoMessage, "%s"); n != 2 {
		t.Fatalf("no-message template placeholders = %d, want 2", n)
	}

	got := fmt.Sprintf(ThirdPartyAPIErrorTemplate, "dji", "312014", "设备升级中")
	want := "第三方接口错误，vendor=dji，code=312014，message=设备升级中"
	if got != want {
		t.Fatalf("template result = %q, want %q", got, want)
	}
}

func TestNewThirdPartyAPIErrorUsesArgsFormatting(t *testing.T) {
	err := NewThirdPartyAPIError("dji", "312014", "设备升级中，请勿重复操作")

	if !IsErrorByPbCode(err, extproto.Code__1_00_THIRD_PARTY_API) {
		t.Fatalf("code = %v, want 100603", status.Code(err))
	}
	if got := status.Code(err); got != codes.Unavailable {
		t.Fatalf("status.Code() = %v, want %v", got, codes.Unavailable)
	}
	want := "第三方接口错误，vendor=dji，code=312014，message=设备升级中，请勿重复操作"
	if got := status.Convert(err).Message(); got != want {
		t.Fatalf("status message = %q, want %q", got, want)
	}
	if int(extproto.Code__1_00_THIRD_PARTY_API) != 100603 {
		t.Fatalf("THIRD_PARTY_API = %d, want 100603", extproto.Code__1_00_THIRD_PARTY_API)
	}
}

func TestNewErrorByPbCodeSupportsTemplateArgs(t *testing.T) {
	direct := NewErrorByPbCode(
		extproto.Code__1_00_THIRD_PARTY_API,
		ThirdPartyAPIErrorTemplate,
		"dji", "312014", "设备升级中",
	)
	wrapped := NewThirdPartyAPIError("dji", "312014", "设备升级中")

	if status.Convert(direct).Message() != status.Convert(wrapped).Message() {
		t.Fatalf("direct = %q, wrapped = %q",
			status.Convert(direct).Message(), status.Convert(wrapped).Message())
	}
}

func TestNewThirdPartyAPIErrorWithoutMessage(t *testing.T) {
	err := NewThirdPartyAPIError("dji", "312014", "")

	want := "第三方接口错误，vendor=dji，code=312014"
	if got := status.Convert(err).Message(); got != want {
		t.Fatalf("status message = %q, want %q", got, want)
	}
}

func TestNewThirdPartyAPIErrorSanitizes(t *testing.T) {
	err := NewThirdPartyAPIError("  dji \n sdk ", " 3120 14 ", "第一行\n第二行\t含  多余空格")

	want := "第三方接口错误，vendor=dji_sdk，code=3120_14，message=第一行 第二行 含 多余空格"
	if got := status.Convert(err).Message(); got != want {
		t.Fatalf("status message = %q, want %q", got, want)
	}

	empty := NewThirdPartyAPIError("", "", "")
	if got := status.Convert(empty).Message(); !strings.Contains(got, "vendor=unknown") || !strings.Contains(got, "code=unknown") {
		t.Fatalf("empty fields should fall back to unknown, got %q", got)
	}
}

func TestNewThirdPartyAPIErrorTruncates(t *testing.T) {
	message := strings.Repeat("错", thirdPartyAPIMaxMsg+50)
	err := NewThirdPartyAPIError("dji", "312014", message)
	got := status.Convert(err).Message()

	if !strings.HasSuffix(got, "…") {
		t.Fatalf("long message should be truncated, got %q", got)
	}
	prefix := "第三方接口错误，vendor=dji，code=312014，message="
	if runes := []rune(got); len(runes) > thirdPartyAPIMaxMsg+len([]rune(prefix))+1 {
		t.Fatalf("truncated message too long: %d runes", len(runes))
	}
}

func TestParseThirdPartyAPIError(t *testing.T) {
	err := NewThirdPartyAPIError("dji", "312014", "设备升级中，请勿重复操作")

	vendor, code, message, ok := ParseThirdPartyAPIError(status.Convert(err).Message())
	if !ok {
		t.Fatalf("ParseThirdPartyAPIError() ok = false")
	}
	if vendor != "dji" || code != "312014" || message != "设备升级中，请勿重复操作" {
		t.Fatalf("parsed = %q/%q/%q", vendor, code, message)
	}
}

func TestParseThirdPartyAPIErrorWithoutMessage(t *testing.T) {
	vendor, code, message, ok := ParseThirdPartyAPIError("第三方接口错误，vendor=oryx，code=500")
	if !ok || vendor != "oryx" || code != "500" || message != "" {
		t.Fatalf("parsed = %q/%q/%q/%v", vendor, code, message, ok)
	}
}

func TestParseThirdPartyAPIErrorRejectsOtherMessages(t *testing.T) {
	for _, msg := range []string{
		"",
		"系统繁忙",
		"第三方接口错误",
		"第三方接口错误，vendor=dji",
		"第三方接口错误，vendor=，code=312014",
		"第三方接口错误，vendor=dji，code=",
		"第三方接口错误（dji/312014）：设备升级中",
	} {
		if _, _, _, ok := ParseThirdPartyAPIError(msg); ok {
			t.Fatalf("ParseThirdPartyAPIError(%q) ok = true, want false", msg)
		}
	}
}
