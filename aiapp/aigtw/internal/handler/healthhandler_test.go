package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aisolo/aisolo"
)

func TestHealthHandlerReturnsReadinessDependencies(t *testing.T) {
	svcCtx := &svc.ServiceContext{AiChatCli: healthFakeAiChatClient{}, AiSoloCli: healthFakeAiSoloClient{}}
	svcCtx.Config.JwtAuth.AccessSecret = "secret"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthHandler(svcCtx).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Status       string            `json:"status"`
		Ready        bool              `json:"ready"`
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "ok" || !body.Ready || body.Version != "aigtw" {
		t.Fatalf("health body = %#v, want ok ready aigtw", body)
	}
	if body.Dependencies["jwt"] != "ok" || body.Dependencies["aichat_rpc"] != "ok" || body.Dependencies["aisolo_rpc"] != "ok" || body.Dependencies["knowledge"] != "disabled" {
		t.Fatalf("dependencies = %#v, want ready deps", body.Dependencies)
	}
}

func TestHealthHandlerReportsMissingReadiness(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthHandler(&svc.ServiceContext{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Ready        bool              `json:"ready"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Ready {
		t.Fatal("ready = true, want false")
	}
	if body.Dependencies["jwt"] != "missing" || body.Dependencies["aichat_rpc"] != "missing" || body.Dependencies["aisolo_rpc"] != "missing" {
		t.Fatalf("dependencies = %#v, want missing deps", body.Dependencies)
	}
}

type healthFakeAiChatClient struct{ aichat.AiChatClient }
type healthFakeAiSoloClient struct{ aisolo.AiSoloClient }
