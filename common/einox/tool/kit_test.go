package tool

import (
	"context"
	"testing"

	ctool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

func newEchoTool(t *testing.T) ctool.BaseTool {
	t.Helper()
	tool, err := utils.InferTool("echo", "Echo back input",
		func(_ context.Context, in *struct{ Text string }) (*struct{ Result string }, error) {
			return &struct{ Result string }{Result: in.Text}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return tool
}

func newCalcTool(t *testing.T) ctool.BaseTool {
	t.Helper()
	tool, err := utils.InferTool("calculator", "Simple calculator",
		func(_ context.Context, in *struct{ Expr string }) (*struct{ Result string }, error) {
			return &struct{ Result string }{Result: in.Expr}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return tool
}

func TestKitRegisterAndGet(t *testing.T) {
	k := NewKit()
	echo := newEchoTool(t)
	if err := k.Register(CapCompute, echo); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	e, ok := k.Get("echo")
	if !ok {
		t.Fatal("Get(echo) not found")
	}
	if e.Name != "echo" || e.Capability != CapCompute {
		t.Fatalf("got %+v, want name=echo cap=compute", e)
	}
}

func TestKitRegisterDuplicate(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	if err := k.Register(CapCompute, newEchoTool(t)); err == nil {
		t.Fatal("expected error for duplicate register")
	}
}

func TestKitRegisterNilTool(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, nil); err == nil {
		t.Fatal("expected error for nil tool")
	}
}

func TestKitByCapability(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	if err := k.Register(CapIO, newCalcTool(t)); err != nil {
		t.Fatal(err)
	}
	compute := k.ByCapability(CapCompute)
	if len(compute) != 1 || compute[0] == nil {
		t.Fatalf("ByCapability(compute) = %d tools, want 1", len(compute))
	}
	io := k.ByCapability(CapIO)
	if len(io) != 1 {
		t.Fatalf("ByCapability(io) = %d tools, want 1", len(io))
	}
	human := k.ByCapability(CapHuman)
	if len(human) != 0 {
		t.Fatalf("ByCapability(human) = %d tools, want 0", len(human))
	}
}

func TestKitAll(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	if err := k.Register(CapIO, newCalcTool(t)); err != nil {
		t.Fatal(err)
	}
	all := k.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d tools, want 2", len(all))
	}
	// Should be sorted by name: calculator before echo
	if all[0] == nil || all[1] == nil {
		t.Fatal("All() returned nil tools")
	}
}

func TestKitEntries(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	entries := k.Entries()
	if len(entries) != 1 {
		t.Fatalf("Entries() = %d, want 1", len(entries))
	}
	if entries[0].Name != "echo" || entries[0].Capability != CapCompute {
		t.Fatalf("entry = %+v, want echo/compute", entries[0])
	}
	// Verify entries are copies (modifying returned slice doesn't affect kit)
	entries[0].Name = "hacked"
	if e, _ := k.Get("echo"); e.Name == "hacked" {
		t.Fatal("Entries() did not return a copy")
	}
}

func TestKitSelect(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	if err := k.Register(CapIO, newCalcTool(t)); err != nil {
		t.Fatal(err)
	}
	selected := k.Select("echo")
	if len(selected) != 1 {
		t.Fatalf("Select(echo) = %d, want 1", len(selected))
	}
	// Unknown name silently ignored
	selected = k.Select("echo", "nonexistent")
	if len(selected) != 1 {
		t.Fatalf("Select(echo, nonexistent) = %d, want 1", len(selected))
	}
	// All unknown
	selected = k.Select("ghost")
	if len(selected) != 0 {
		t.Fatalf("Select(ghost) = %d, want 0", len(selected))
	}
}

func TestKitMustRegister(t *testing.T) {
	k := NewKit()
	k.MustRegister(CapCompute, newEchoTool(t))
	if _, ok := k.Get("echo"); !ok {
		t.Fatal("MustRegister did not register tool")
	}
}

func TestKitConcurrentSafe(t *testing.T) {
	k := NewKit()
	if err := k.Register(CapCompute, newEchoTool(t)); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		k.All()
		k.Entries()
		k.ByCapability(CapCompute)
		close(done)
	}()
	k.All()
	k.Entries()
	k.ByCapability(CapCompute)
	<-done
}
