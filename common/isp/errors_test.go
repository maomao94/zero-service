package isp

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestResponseCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: StatusSuccess},
		{name: "retry", err: fmt.Errorf("wrapped: %w", ErrRetry), want: StatusRetry},
		{name: "reject", err: ErrReject, want: StatusReject},
		{name: "internal", err: errors.New("unexpected"), want: StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResponseCode(tt.err); got != tt.want {
				t.Fatalf("ResponseCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIspErrorsSupportErrorsIsAndAs(t *testing.T) {
	wrapped := fmt.Errorf("handler failed: %w", ErrUnimplemented)
	if !errors.Is(wrapped, ErrUnimplemented) {
		t.Fatal("wrapped error should match ErrUnimplemented")
	}

	var ispErr *IspError
	if !errors.As(wrapped, &ispErr) {
		t.Fatal("wrapped error should expose IspError")
	}
	if ispErr.Code != StatusError {
		t.Fatalf("IspError.Code = %q, want %q", ispErr.Code, StatusError)
	}
}

func TestClientErrorsAreSentinels(t *testing.T) {
	errs := []error{
		ErrInvalidMessageType,
		ErrClientNotRegistered,
		ErrSessionUnavailable,
		ErrRequestFailed,
		ErrUnexpectedResponse,
	}
	for _, err := range errs {
		if err == nil || err.Error() == "" {
			t.Fatalf("invalid client error: %v", err)
		}
	}
}

func TestClientExecuteReturnsPackageErrors(t *testing.T) {
	if _, err := (&Client{}).Execute(t.Context(), 0, 1, "", nil); !errors.Is(err, ErrInvalidMessageType) {
		t.Fatalf("invalid message type error = %v, want ErrInvalidMessageType", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	client, err := NewClient(ClientConfig{
		ServerAddr:        addr,
		RootName:          RootPatrolDevice,
		ReconnectInterval: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if _, err := client.Execute(t.Context(), 1, 1, "", nil); !errors.Is(err, ErrClientNotRegistered) {
		t.Fatalf("unregistered error = %v, want ErrClientNotRegistered", err)
	}
}
