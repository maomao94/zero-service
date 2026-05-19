package solo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zero-service/aiapp/aigtw/internal/svc"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestChatHandlerRejectsInvalidRequestBeforeSSEHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/solo/v1/chat", strings.NewReader(`{"sessionId":"sess-1","message":"   "}`))
	req.Header.Set("Content-Type", "application/json")

	ChatHandler(&svc.ServiceContext{}).ServeHTTP(rec, req)

	assertNoSSEHeaders(t, rec, "message is required")
}

func TestResumeHandlerRejectsInvalidRequestBeforeSSEHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/solo/v1/resume/interrupt-1", strings.NewReader(`{"sessionId":"sess-1"}`))
	req = pathvar.WithVars(req, map[string]string{"interruptId": "interrupt-1"})
	req.Header.Set("Content-Type", "application/json")

	ResumeHandler(&svc.ServiceContext{}).ServeHTTP(rec, req)

	assertNoSSEHeaders(t, rec, "action must be yes or no")
}

func assertNoSSEHeaders(t *testing.T, rec *httptest.ResponseRecorder, wantBody string) {
	t.Helper()
	if rec.Code == http.StatusOK {
		t.Fatalf("status = 200, want validation error before SSE success")
	}
	if got := rec.Header().Get("Content-Type"); strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want non-SSE validation response", got)
	}
	if got := rec.Header().Get("X-Accel-Buffering"); got != "" {
		t.Fatalf("X-Accel-Buffering = %q, want empty before SSE headers", got)
	}
	if got := rec.Body.String(); !strings.Contains(got, wantBody) {
		t.Fatalf("body = %q, want containing %q", got, wantBody)
	}
}
