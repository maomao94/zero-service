package svc

import (
	"testing"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aisolo/aisolo"
	einoxkb "zero-service/common/einox/knowledge"
)

func TestServiceContextCloseIsNilSafe(t *testing.T) {
	ctx := &ServiceContext{}

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestServiceContextCloseClosesKnowledge(t *testing.T) {
	kb, err := einoxkb.NewService(einoxkb.Config{Enabled: true, Backend: "memory"}, "test-key")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	ctx := &ServiceContext{Knowledge: kb}

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestDependenciesReportReadyGateway(t *testing.T) {
	ctx := &ServiceContext{
		Config:    config.Config{},
		AiChatCli: fakeAiChatClient{},
		AiSoloCli: fakeAiSoloClient{},
	}
	ctx.Config.JwtAuth.AccessSecret = "secret"

	deps := ctx.Dependencies()
	if deps["jwt"] != "ok" || deps["aichat_rpc"] != "ok" || deps["aisolo_rpc"] != "ok" || deps["knowledge"] != "disabled" {
		t.Fatalf("dependencies = %#v, want ready base deps", deps)
	}
	if !ctx.Ready() {
		t.Fatal("Ready() = false, want true")
	}
}

func TestDependenciesReportMissingCoreDeps(t *testing.T) {
	ctx := &ServiceContext{}
	deps := ctx.Dependencies()
	if deps["jwt"] != "missing" || deps["aichat_rpc"] != "missing" || deps["aisolo_rpc"] != "missing" {
		t.Fatalf("dependencies = %#v, want missing core deps", deps)
	}
	if ctx.Ready() {
		t.Fatal("Ready() = true, want false for missing core deps")
	}
}

func TestDependenciesReportKnowledgeStates(t *testing.T) {
	misconfigured := &ServiceContext{
		Config:           config.Config{},
		AiChatCli:        fakeAiChatClient{},
		AiSoloCli:        fakeAiSoloClient{},
		KnowledgeInitErr: "embedding key missing",
	}
	misconfigured.Config.JwtAuth.AccessSecret = "secret"
	misconfigured.Config.Knowledge.Enabled = true
	misconfigured.Config.Knowledge.Backend = "memory"

	deps := misconfigured.Dependencies()
	if deps["knowledge"] != "misconfigured" || deps["knowledge_backend"] != "memory" || deps["knowledge_error"] != "embedding key missing" {
		t.Fatalf("dependencies = %#v, want misconfigured knowledge", deps)
	}
	if misconfigured.Ready() {
		t.Fatal("Ready() = true, want false for misconfigured knowledge")
	}

	misconfigured.Knowledge = &einoxkb.Service{}
	deps = misconfigured.Dependencies()
	if deps["knowledge"] != "ok" || deps["knowledge_error"] != "" {
		t.Fatalf("dependencies = %#v, want ok knowledge without error", deps)
	}
	if !misconfigured.Ready() {
		t.Fatal("Ready() = false, want true with knowledge ready")
	}
}

type fakeAiChatClient struct{ aichat.AiChatClient }
type fakeAiSoloClient struct{ aisolo.AiSoloClient }
