package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
}

func TestGlobalMetrics(t *testing.T) {
	m := Global()
	if m == nil {
		t.Fatal("Global() returned nil")
	}
	m2 := Global()
	if m != m2 {
		t.Fatal("Global() should return singleton")
	}
}

func TestRecordTurn(t *testing.T) {
	m := NewMetrics()
	m.RecordTurn(context.Background(), "agent", "ok", time.Second)
}

func TestRecordAgent(t *testing.T) {
	m := NewMetrics()
	m.RecordAgent(context.Background(), "chat", "ok", time.Millisecond*500)
}

func TestRecordToolCall(t *testing.T) {
	m := NewMetrics()
	m.RecordToolCall(context.Background(), "echo", "ok", time.Millisecond*100)
	m.RecordToolCall(context.Background(), "calculator", "error", time.Millisecond*200)
}

func TestRecordInterrupt(t *testing.T) {
	m := NewMetrics()
	m.RecordInterrupt(context.Background(), "approval", "sensitive_tool", "interrupt-001")
	m.RecordInterrupt(context.Background(), "free_text", "ask_input", "")
}

func TestRecordResume(t *testing.T) {
	m := NewMetrics()
	m.RecordResume(context.Background(), "approval", "ok", "agent", "interrupt-001", "yes", time.Second)
	m.RecordResume(context.Background(), "select", "interrupted_again", "deep", "interrupt-002", "", time.Second*2)
}

func TestRecordCheckPoint(t *testing.T) {
	m := NewMetrics()
	m.RecordCheckPoint(context.Background(), "set", "ok")
	m.RecordCheckPoint(context.Background(), "get", "error")
}

func TestRecordKnowledge(t *testing.T) {
	m := NewMetrics()
	m.RecordKnowledge(context.Background(), "search", "ok", "memory", time.Millisecond*300)
	m.RecordKnowledge(context.Background(), "ingest", "error", "milvus", time.Second*5)
}

func TestPromLabelKindOrTool(t *testing.T) {
	if got := promLabelKindOrTool(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := promLabelKindOrTool("normal"); got != "normal" {
		t.Fatalf("normal: got %q", got)
	}
	if got := promLabelKindOrTool("a\nb"); got != "invalid_enum_string" {
		t.Fatalf("control char: got %q", got)
	}
	if got := promLabelKindOrTool(string([]byte{0xff, 0xfe})); got != "invalid_utf8" {
		t.Fatalf("invalid utf8: got %q", got)
	}
}

func TestMultipleMetricsInstances(t *testing.T) {
	m1 := NewMetrics()
	m2 := NewMetrics()
	// Both should work without panicking
	m1.RecordTurn(context.Background(), "agent", "ok", time.Second)
	m2.RecordTurn(context.Background(), "deep", "error", time.Second)
}

func TestLabelSanitizationEdgeCases(t *testing.T) {
	m := NewMetrics()
	// Empty sanitized labels should not cause panics
	m.RecordKnowledge(context.Background(), "search", "ok", "", time.Second)
	m.RecordResume(context.Background(), "", "ok", "", "id-1", "", time.Second)
}

func TestRecordAgentSlowLog(t *testing.T) {
	m := NewMetrics()
	m.RecordAgent(context.Background(), "chat", "ok", time.Second*2)
}
