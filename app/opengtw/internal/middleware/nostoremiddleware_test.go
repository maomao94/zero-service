package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoStoreMiddleware(t *testing.T) {
	handler := NewNoStoreMiddleware().Handle(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/open-admin/v1/app/sign-key/reveal", nil))

	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rec.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want no-cache", got)
	}
}
