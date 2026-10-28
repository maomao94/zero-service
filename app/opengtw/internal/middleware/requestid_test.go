package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Run("preserves incoming id", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/v1/gateway", nil)
		request.Header.Set("X-Request-Id", "req-from-client")
		recorder := httptest.NewRecorder()

		RequestIDMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if got := RequestIDFromContext(r.Context()); got != "req-from-client" {
				t.Fatalf("context request id = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		})(recorder, request)

		if got := recorder.Header().Get("X-Request-Id"); got != "req-from-client" {
			t.Fatalf("response request id = %q", got)
		}
	})

	t.Run("generates missing id", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/v1/gateway", nil)
		recorder := httptest.NewRecorder()

		RequestIDMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if got := RequestIDFromContext(r.Context()); got == "" {
				t.Fatal("context request id is empty")
			}
			w.WriteHeader(http.StatusNoContent)
		})(recorder, request)

		if got := recorder.Header().Get("X-Request-Id"); got == "" {
			t.Fatal("response request id is empty")
		}
	})
}
