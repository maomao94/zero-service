package broadcast

import (
	"errors"
	"fmt"
	"testing"

	"zero-service/common/antsx"
)

func TestNormalizeErrorKindBuiltins(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: antsx.ErrReplyExpired, want: KindTimeout},
		{name: "duplicate", err: antsx.ErrDuplicateID, want: KindDuplicate},
		{name: "nil", err: nil, want: ""},
		{name: "unknown", err: errors.New("boom"), want: KindUnknown},
		{name: "unknown_wrapped", err: fmt.Errorf("wrapped: %w", errors.New("boom")), want: KindUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeErrorKind(tt.err); got != tt.want {
				t.Fatalf("NormalizeErrorKind(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorFromKindBuiltins(t *testing.T) {
	if err := ErrorFromKind(KindTimeout, "msg"); !errors.Is(err, antsx.ErrReplyExpired) {
		t.Fatalf("ErrorFromKind(timeout) = %v, want ErrReplyExpired", err)
	}
	if err := ErrorFromKind(KindDuplicate, "msg"); !errors.Is(err, antsx.ErrDuplicateID) {
		t.Fatalf("ErrorFromKind(duplicate) = %v, want ErrDuplicateID", err)
	}
	err := ErrorFromKind(KindUnknown, "some error text")
	if err == nil || err.Error() != "some error text" {
		t.Fatalf("ErrorFromKind(unknown) = %v, want text error", err)
	}
}

func TestRegisterErrorKindRoundtrip(t *testing.T) {
	src := errors.New("business rejected")
	kind := "business_rejected"
	RegisterErrorKind(src, kind, func(msg string) error {
		return fmt.Errorf("restored: %s", msg)
	})

	if got := NormalizeErrorKind(src); got != kind {
		t.Fatalf("NormalizeErrorKind(src) = %q, want %q", got, kind)
	}
	wrapped := fmt.Errorf("wrap: %w", src)
	if got := NormalizeErrorKind(wrapped); got != kind {
		t.Fatalf("NormalizeErrorKind(wrapped) = %q, want %q", got, kind)
	}

	restored := ErrorFromKind(kind, "device says no")
	if restored == nil || restored.Error() != "restored: device says no" {
		t.Fatalf("ErrorFromKind(kind) = %v, want restored error", restored)
	}
}
