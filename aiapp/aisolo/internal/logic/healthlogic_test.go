package logic

import (
	"context"
	"testing"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/einox/tool/builtin"
)

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
