package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestApprovalMiddlewareSkipsNonTargetTools(t *testing.T) {
	m := NewApprovalMiddleware([]string{"sensitive_tool"})
	called := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		called = true
		return "done", nil
	}

	ep, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "safe_tool"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}
	result, err := ep(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	if !called {
		t.Fatal("endpoint was not called for non-target tool")
	}
	if result != "done" {
		t.Fatalf("got %q, want %q", result, "done")
	}
}

func TestApprovalMiddlewareInterruptsOnFirstCall(t *testing.T) {
	m := NewApprovalMiddleware([]string{"sensitive_tool"})
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "done", nil
	}

	ep, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "sensitive_tool"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}

	_, err = ep(context.Background(), `{"key":"value"}`)
	if err == nil {
		t.Fatal("expected interrupt error on first call, got nil")
	}
}

func TestApprovalMiddlewareWithCustomConfig(t *testing.T) {
	m := NewApprovalMiddleware([]string{"tool_a", "tool_b"}).
		WithApprovalConfig("tool_a", &ApprovalConfig{Question: "确认执行？"})

	// First call on tool_a should interrupt (not call endpoint)
	ep, err := m.WrapInvokableToolCall(context.Background(),
		func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "ok", nil
		},
		&adk.ToolContext{Name: "tool_a"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}

	_, err = ep(context.Background(), `{}`)
	if err == nil {
		t.Fatal("expected interrupt error for tool_a")
	}

	// tool_b should also interrupt (in approval set, no custom config)
	ep2, err := m.WrapInvokableToolCall(context.Background(),
		func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "ok", nil
		},
		&adk.ToolContext{Name: "tool_b"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}
	_, err = ep2(context.Background(), `{}`)
	if err == nil {
		t.Fatal("expected interrupt error for tool_b")
	}
}

func TestApprovalMiddlewareStreamSkipsNonTarget(t *testing.T) {
	m := NewApprovalMiddleware([]string{"sensitive_tool"})
	called := false

	streamEp, err := m.WrapStreamableToolCall(context.Background(),
		func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			called = true
			return schema.StreamReaderFromArray([]string{"stream-ok"}), nil
		},
		&adk.ToolContext{Name: "safe_tool"})
	if err != nil {
		t.Fatalf("WrapStreamableToolCall error = %v", err)
	}

	reader, err := streamEp(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("stream invoke error = %v", err)
	}
	if !called {
		t.Fatal("endpoint was not called for non-target tool")
	}
	chunk, err := reader.Recv()
	if err != nil || chunk != "stream-ok" {
		t.Fatalf("got chunk=%q err=%v, want stream-ok", chunk, err)
	}
}

func TestApprovalMiddlewareStreamInterruptsOnFirstCall(t *testing.T) {
	m := NewApprovalMiddleware([]string{"sensitive_tool"})

	streamEp, err := m.WrapStreamableToolCall(context.Background(),
		func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			return schema.StreamReaderFromArray([]string{"ok"}), nil
		},
		&adk.ToolContext{Name: "sensitive_tool"})
	if err != nil {
		t.Fatalf("WrapStreamableToolCall error = %v", err)
	}

	_, err = streamEp(context.Background(), `{}`)
	if err == nil {
		t.Fatal("expected interrupt error on first stream call, got nil")
	}
}

func TestApprovalMiddlewareEmptyToolList(t *testing.T) {
	m := NewApprovalMiddleware(nil)
	called := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		called = true
		return "result", nil
	}

	ep, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "any_tool"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}
	result, err := ep(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	if !called {
		t.Fatal("endpoint was not called with empty tool list")
	}
	if result != "result" {
		t.Fatalf("got %q, want %q", result, "result")
	}
}

func TestSingleChunkReader(t *testing.T) {
	reader := singleChunkReader("hello")
	chunk, err := reader.Recv()
	if err != nil || chunk != "hello" {
		t.Fatalf("got chunk=%q err=%v", chunk, err)
	}
	_, err = reader.Recv()
	if err == nil {
		t.Fatal("expected EOF on second recv")
	}
	reader.Close()
}

func TestApprovalMiddlewareAllowsAllWithoutTargetList(t *testing.T) {
	m := NewApprovalMiddleware(nil)
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "allowed", nil
	}

	st := m.WithApprovalConfig("some_tool", &ApprovalConfig{Question: "?"})
	if st != m {
		t.Fatal("WithApprovalConfig should return self")
	}

	ep, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "some_tool"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall error = %v", err)
	}
	result, err := ep(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	if result != "allowed" {
		t.Fatalf("got %q, want %q", result, "allowed")
	}
}
