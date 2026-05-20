package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"zero-service/common/einox/protocol"
)

func TestToolRegistryListsStaticToolInfo(t *testing.T) {
	registry, err := NewToolRegistry(StaticTool{Name: "read_allowed_file", Desc: "Read an allowed file", Result: "ok"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}

	infos, err := registry.Infos(context.Background())
	if err != nil {
		t.Fatalf("Infos() error = %v", err)
	}
	if len(infos) != 1 || infos[0].Name != "read_allowed_file" {
		t.Fatalf("tool infos = %#v, want read_allowed_file", infos)
	}
}

func TestToolRegistryInfosPreservesRegistrationOrder(t *testing.T) {
	registry, err := NewToolRegistry(
		StaticTool{Name: "z_last", Desc: "last"},
		StaticTool{Name: "a_first", Desc: "first"},
		StaticTool{Name: "m_middle", Desc: "middle"},
	)
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}

	infos, err := registry.Infos(context.Background())
	if err != nil {
		t.Fatalf("Infos() error = %v", err)
	}
	if got := toolInfoNames(infos); strings.Join(got, ",") != "z_last,a_first,m_middle" {
		t.Fatalf("tool info names = %#v, want registration order", got)
	}
}

func TestToolRegistryRejectsDuplicateTool(t *testing.T) {
	_, err := NewToolRegistry(
		StaticTool{Name: "read_allowed_file", Desc: "Read an allowed file"},
		StaticTool{Name: "read_allowed_file", Desc: "Duplicate"},
	)
	if err == nil {
		t.Fatal("NewToolRegistry() error = nil, want duplicate error")
	}
}

func TestToolRegistryRejectsNilTool(t *testing.T) {
	_, err := NewToolRegistry(nil)
	if err == nil || !strings.Contains(err.Error(), "nil tool") {
		t.Fatalf("NewToolRegistry(nil) error = %v, want nil tool error", err)
	}
}

func TestToolRegistryRejectsEmptyToolName(t *testing.T) {
	_, err := NewToolRegistry(StaticTool{Desc: "missing name"})
	if err == nil || !strings.Contains(err.Error(), "tool info name is empty") {
		t.Fatalf("NewToolRegistry(empty name) error = %v, want empty name error", err)
	}
}

func TestToolRegistryRunInvokableTool(t *testing.T) {
	calls := &ToolCalls{}
	registry, err := NewToolRegistry(StaticTool{Name: "echo", Desc: "Echo args", Result: "ok", Calls: calls})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}

	result, err := registry.Run(context.Background(), "echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result != "ok" {
		t.Fatalf("Run() result = %q, want ok", result)
	}
	if len(calls.Args) != 1 || calls.Args[0] != `{"text":"hello"}` {
		t.Fatalf("tool args = %#v, want original JSON args", calls.Args)
	}
}

func TestToolRegistryRunWithEmitterEmitsToolEvents(t *testing.T) {
	registry, err := NewToolRegistry(StaticTool{Name: "echo", Desc: "Echo args", Result: "ok"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	collector := newEventCollector(Request{SessionID: "session-test-001", TurnID: "turn-test-001"})
	em := protocol.NewEmitter(collector, collector.sessionID, collector.turnID)

	result, err := registry.RunWithEmitter(context.Background(), em, "echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("RunWithEmitter() error = %v", err)
	}
	if result != "ok" {
		t.Fatalf("RunWithEmitter() result = %q, want ok", result)
	}
	assertEventTypes(t, collector.events, protocol.EventToolCallStart, protocol.EventToolCallEnd)
	end, ok := decodeData[protocol.ToolCallEndData](t, collector.events[1])
	if !ok || end.Tool != "echo" || end.Result != "ok" || end.Error != "" {
		t.Fatalf("tool end = %#v, want echo ok", end)
	}
}

func TestToolRegistryRunWithEmitterCallIDUsesProvidedID(t *testing.T) {
	registry, err := NewToolRegistry(StaticTool{Name: "echo", Desc: "Echo args", Result: "ok"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	collector := newEventCollector(Request{SessionID: "session-test-001", TurnID: "turn-test-001"})
	em := protocol.NewEmitter(collector, collector.sessionID, collector.turnID)

	result, err := registry.RunWithEmitterCallID(context.Background(), em, "model-call-1", "echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("RunWithEmitterCallID() error = %v", err)
	}
	if result != "ok" {
		t.Fatalf("RunWithEmitterCallID() result = %q, want ok", result)
	}
	assertEventTypes(t, collector.events, protocol.EventToolCallStart, protocol.EventToolCallEnd)
	start, ok := decodeData[protocol.ToolCallStartData](t, collector.events[0])
	if !ok || start.CallID != "model-call-1" {
		t.Fatalf("tool start = %#v, want model-call-1", start)
	}
	end, ok := decodeData[protocol.ToolCallEndData](t, collector.events[1])
	if !ok || end.CallID != "model-call-1" {
		t.Fatalf("tool end = %#v, want model-call-1", end)
	}
}

func TestToolRegistryRunWithEmitterEmitsToolError(t *testing.T) {
	want := errors.New("tool failed")
	registry, err := NewToolRegistry(StaticTool{Name: "fail", Err: want})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	collector := newEventCollector(Request{SessionID: "session-test-001", TurnID: "turn-test-001"})
	em := protocol.NewEmitter(collector, collector.sessionID, collector.turnID)

	_, err = registry.RunWithEmitter(context.Background(), em, "fail", `{}`)
	if !errors.Is(err, want) {
		t.Fatalf("RunWithEmitter() error = %v, want %v", err, want)
	}
	assertEventTypes(t, collector.events, protocol.EventToolCallStart, protocol.EventToolCallEnd)
	end, ok := decodeData[protocol.ToolCallEndData](t, collector.events[1])
	if !ok || end.Tool != "fail" || !strings.Contains(end.Error, "tool failed") {
		t.Fatalf("tool end = %#v, want fail error", end)
	}
}

func TestToolRegistryRunReturnsToolError(t *testing.T) {
	want := errors.New("tool failed")
	registry, err := NewToolRegistry(StaticTool{Name: "fail", Err: want})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}

	_, err = registry.Run(context.Background(), "fail", `{}`)
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
}

func TestToolRegistryRunRejectsMissingTool(t *testing.T) {
	registry, err := NewToolRegistry()
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}

	_, err = registry.Run(context.Background(), "missing", `{}`)
	if err == nil {
		t.Fatal("Run() error = nil, want missing tool error")
	}
}

func TestStaticRetrieverAndDocumentsContext(t *testing.T) {
	retriever := StaticRetriever{Documents: []*schema.Document{
		{ID: "doc-1", Content: "Eino is a Go framework for LLM applications."},
		{ID: "doc-2", Content: "It provides components, compose, and ADK."},
	}}

	docs, err := retriever.Retrieve(context.Background(), "What is Eino?", 3)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	ctx := DocumentsContext(docs)
	if !strings.Contains(ctx, "[1] Eino is a Go framework") {
		t.Fatalf("context missing first citation: %q", ctx)
	}
	if !strings.Contains(ctx, "[2] It provides components") {
		t.Fatalf("context missing second citation: %q", ctx)
	}
}

func toolInfoNames(infos []*schema.ToolInfo) []string {
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if info != nil {
			names = append(names, info.Name)
		}
	}
	return names
}
