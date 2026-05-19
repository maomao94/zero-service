package solo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aisolo/aisolo"
)

func TestGatewayMetaHandlerReturnsReadinessDependencies(t *testing.T) {
	svcCtx := &svc.ServiceContext{AiChatCli: metaFakeAiChatClient{}, AiSoloCli: metaFakeAiSoloClient{}}
	svcCtx.Config.JwtAuth.AccessSecret = "secret"
	svcCtx.Config.Knowledge.Enabled = true
	svcCtx.Config.Knowledge.Backend = "memory"
	svcCtx.KnowledgeInitErr = "embedding key missing"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/solo/v1/meta", nil)

	GatewayMetaHandler(svcCtx).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Ready            bool              `json:"ready"`
		Dependencies     map[string]string `json:"dependencies"`
		KnowledgeBackend string            `json:"knowledgeBackend"`
		Knowledge        string            `json:"knowledge"`
		KnowledgeError   string            `json:"knowledge_error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode meta response: %v", err)
	}
	if body.Ready {
		t.Fatal("ready = true, want false for misconfigured knowledge")
	}
	if body.KnowledgeBackend != "memory" || body.Knowledge != "misconfigured" || body.KnowledgeError != "embedding key missing" {
		t.Fatalf("meta body = %#v, want knowledge compatibility fields", body)
	}
	if body.Dependencies["knowledge"] != "misconfigured" || body.Dependencies["knowledge_backend"] != "memory" || body.Dependencies["knowledge_error"] != "embedding key missing" {
		t.Fatalf("dependencies = %#v, want knowledge dependency details", body.Dependencies)
	}
}

type metaFakeAiChatClient struct{ aichat.AiChatClient }
type metaFakeAiSoloClient struct{ aisolo.AiSoloClient }
