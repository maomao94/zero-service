package tool

import (
	"context"
	"testing"

	ctool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

func newTool(t *testing.T, name, desc string) ctool.BaseTool {
	t.Helper()
	tool, err := utils.InferTool(name, desc,
		func(_ context.Context, in *struct{}) (*struct{}, error) {
			return &struct{}{}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return tool
}

func kitWithTools(t *testing.T, entries map[Capability][]string) *Kit {
	t.Helper()
	k := NewKit()
	for cap, names := range entries {
		for _, name := range names {
			if err := k.Register(cap, newTool(t, name, "tool "+name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	return k
}

func TestPolicyOpenAllowsAll(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo", "calc"},
		CapIO:      {"now"},
	})
	p := NewPolicy()
	tools := p.Apply(k)
	if len(tools) != 3 {
		t.Fatalf("open policy: got %d tools, want 3", len(tools))
	}
}

func TestPolicyNilAllowsAll(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo"},
	})
	var p *Policy
	tools := p.Apply(k)
	if len(tools) != 1 {
		t.Fatalf("nil policy: got %d tools, want 1", len(tools))
	}
}

func TestPolicyAllowCapability(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo", "calc"},
		CapIO:      {"now", "random_id"},
	})
	p := NewPolicy().AllowCapabilities(CapCompute)
	tools := p.Apply(k)
	if len(tools) != 2 {
		t.Fatalf("allow compute only: got %d tools, want 2", len(tools))
	}
}

func TestPolicyAllowMultipleCapabilities(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo"},
		CapIO:      {"now"},
		CapHuman:   {"ask_confirm"},
	})
	p := NewPolicy().AllowCapabilities(CapCompute, CapHuman)
	tools := p.Apply(k)
	if len(tools) != 2 {
		t.Fatalf("allow compute+human: got %d tools, want 2", len(tools))
	}
}

func TestPolicyAllowNames(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo", "calc"},
		CapIO:      {"now"},
	})
	p := NewPolicy().AllowNames("echo", "now")
	tools := p.Apply(k)
	if len(tools) != 2 {
		t.Fatalf("allow names echo+now: got %d tools, want 2", len(tools))
	}
}

func TestPolicyAllowCapabilityAndName(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo", "calc"},
		CapIO:      {"now"},
		CapHuman:   {"ask_confirm"},
	})
	p := NewPolicy().AllowCapabilities(CapCompute).AllowNames("now")
	tools := p.Apply(k)
	if len(tools) != 3 {
		t.Fatalf("allow compute+now: got %d tools, want 3", len(tools))
	}
}

func TestPolicyEmptyKit(t *testing.T) {
	p := NewPolicy().AllowCapabilities(CapCompute)
	tools := p.Apply(NewKit())
	if len(tools) != 0 {
		t.Fatalf("empty kit: got %d tools, want 0", len(tools))
	}
}

func TestPolicyKitIsNil(t *testing.T) {
	p := NewPolicy().AllowCapabilities(CapCompute)
	if tools := p.Apply(nil); tools != nil {
		t.Fatal("nil kit should return nil")
	}
}

func TestPolicyAllowsCapabilityAcrossNames(t *testing.T) {
	k := kitWithTools(t, map[Capability][]string{
		CapCompute: {"echo"},
		CapIO:      {"now"},
	})
	// AllowNames with a name from a capability not in AllowCapabilities
	p := NewPolicy().AllowCapabilities(CapCompute).AllowNames("now")
	tools := p.Apply(k)
	if len(tools) != 2 {
		t.Fatalf("allow compute+name now: got %d tools, want 2", len(tools))
	}
}
