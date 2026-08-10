package djisdk

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNewDJIErrorKnownCode(t *testing.T) {
	err := NewDJIError(312022)
	if err.Code != 312022 {
		t.Fatalf("Code = %d, want 312022", err.Code)
	}
	if err.Name != "DJI_AIRCRAFT_START_FAILED_DISCONNECTED" {
		t.Fatalf("Name = %q", err.Name)
	}
	if err.Message == "" || err.Message == err.Name {
		t.Fatalf("Message = %q, want non-empty Chinese description", err.Message)
	}
}

func TestNewDJIErrorUnknownCode(t *testing.T) {
	err := NewDJIError(999999)
	if err.Name != "UNKNOWN(999999)" {
		t.Fatalf("Name = %q, want UNKNOWN(999999)", err.Name)
	}
	if err.Message != err.Name {
		t.Fatalf("Message = %q, want fallback to Name", err.Message)
	}
}

func TestDJIErrorMessage(t *testing.T) {
	err := NewDJIError(312022)
	msg := err.Error()
	if !strings.Contains(msg, "code=312022") || !strings.Contains(msg, err.Message) {
		t.Fatalf("Error() = %q", msg)
	}
}

func TestIsDJIError(t *testing.T) {
	direct := NewDJIError(312022)
	if got, ok := IsDJIError(direct); !ok || got != direct {
		t.Fatalf("IsDJIError(direct) = %+v, %v", got, ok)
	}

	wrapped := fmt.Errorf("command failed: %w", direct)
	if got, ok := IsDJIError(wrapped); !ok || got.Code != 312022 {
		t.Fatalf("IsDJIError(wrapped) = %+v, %v", got, ok)
	}

	if _, ok := IsDJIError(errors.New("plain error")); ok {
		t.Fatal("IsDJIError(plain) = true, want false")
	}
	if _, ok := IsDJIError(nil); ok {
		t.Fatal("IsDJIError(nil) = true, want false")
	}
}

func TestPlatformErrorUnwrapAndResultFromError(t *testing.T) {
	base := errors.New("handler failed")

	t.Run("explicit platform error", func(t *testing.T) {
		pe := &PlatformError{Code: PlatformResultTimeout, Err: base}
		if !strings.Contains(pe.Error(), "code=2") {
			t.Fatalf("Error() = %q", pe.Error())
		}
		if !errors.Is(pe, base) {
			t.Fatal("errors.Is(PlatformError) did not match underlying error")
		}
		if got := ResultFromError(pe); got != PlatformResultTimeout {
			t.Fatalf("ResultFromError() = %d, want %d", got, PlatformResultTimeout)
		}
	})

	t.Run("wrapped platform error", func(t *testing.T) {
		pe := &PlatformError{Code: PlatformResultOK, Err: base}
		wrapped := fmt.Errorf("outer: %w", pe)
		if got := ResultFromError(wrapped); got != PlatformResultOK {
			t.Fatalf("ResultFromError(wrapped) = %d, want %d", got, PlatformResultOK)
		}
	})

	t.Run("default handler error", func(t *testing.T) {
		if got := ResultFromError(base); got != PlatformResultHandlerError {
			t.Fatalf("ResultFromError(plain) = %d, want %d", got, PlatformResultHandlerError)
		}
		if got := ResultFromError(nil); got != PlatformResultHandlerError {
			t.Fatalf("ResultFromError(nil) = %d, want %d", got, PlatformResultHandlerError)
		}
	})
}

func TestErrSkipRequestReplyIsSentinel(t *testing.T) {
	if ErrSkipRequestReply == nil {
		t.Fatal("ErrSkipRequestReply is nil")
	}
	if !errors.Is(ErrSkipRequestReply, ErrSkipRequestReply) {
		t.Fatal("sentinel must be self-identifiable via errors.Is")
	}
}
