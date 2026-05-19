package logic

import (
	"context"
	"testing"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	einoxruntime "zero-service/common/einox/runtime"
	"zero-service/common/einox/tool/builtin"
)

func TestHealthReportsMissingRuntimeDependencies(t *testing.T) {
	resp, err := NewHealthLogic(context.Background(), &svc.ServiceContext{}).Health(&aisolo.HealthReq{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if resp.GetStatus() != "ok" || resp.GetVersion() != "refactor" || resp.GetTimestamp() <= 0 {
		t.Fatalf("Health() = %#v, want ok refactor with timestamp", resp)
	}
	deps := resp.GetDependencies()
	for _, key := range []string{"chat_model", "tool_kit", "runtime_runner", "runtime_tools", "executor"} {
		if deps[key] != "missing" {
			t.Fatalf("%s = %q, want missing in deps %#v", key, deps[key], deps)
		}
	}
	if deps["knowledge"] != "disabled" {
		t.Fatalf("knowledge = %q, want disabled", deps["knowledge"])
	}
}

func TestHealthReportsToolKitStatus(t *testing.T) {
	ctx := &svc.ServiceContext{Kit: builtin.MustNewDefaultKit()}

	resp, err := NewHealthLogic(context.Background(), ctx).Health(&aisolo.HealthReq{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got := resp.Dependencies["tool_kit"]; got != "ok" {
		t.Fatalf("tool_kit = %q, want ok", got)
	}
}

func TestHealthReportsToolKitInitError(t *testing.T) {
	ctx := &svc.ServiceContext{KitInitErr: "construct failed"}

	resp, err := NewHealthLogic(context.Background(), ctx).Health(&aisolo.HealthReq{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got := resp.Dependencies["tool_kit"]; got != "missing" {
		t.Fatalf("tool_kit = %q, want missing", got)
	}
	if got := resp.Dependencies["tool_kit_error"]; got != "construct failed" {
		t.Fatalf("tool_kit_error = %q, want construct failed", got)
	}
}

func TestHealthReportsKnowledgeMisconfigured(t *testing.T) {
	ctx := &svc.ServiceContext{KnowledgeInitErr: "embedding key missing"}
	ctx.Config.Knowledge.Enabled = true
	ctx.Config.Knowledge.Backend = "memory"

	resp, err := NewHealthLogic(context.Background(), ctx).Health(&aisolo.HealthReq{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	deps := resp.GetDependencies()
	if deps["knowledge"] != "misconfigured" || deps["knowledge_backend"] != "memory" || deps["knowledge_error"] != "embedding key missing" {
		t.Fatalf("dependencies = %#v, want misconfigured knowledge", deps)
	}
}

func TestHealthReportsRuntimeToolsReady(t *testing.T) {
	registry, err := einoxruntime.NewToolRegistry()
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	ctx := &svc.ServiceContext{RuntimeTools: registry}

	resp, err := NewHealthLogic(context.Background(), ctx).Health(&aisolo.HealthReq{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got := resp.GetDependencies()["runtime_tools"]; got != "ok" {
		t.Fatalf("runtime_tools = %q, want ok", got)
	}
}
