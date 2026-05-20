package tool

import (
	"context"
	"testing"
)

type mockCaller struct {
	result string
	err    error
}

func (m *mockCaller) CallTool(_ context.Context, name string, _ map[string]any) (string, error) {
	return m.result + ":" + name, m.err
}

func TestMCPToolInfo(t *testing.T) {
	mt := NewMCPTool(&mockCaller{}, "my_tool", "my description")
	info, err := mt.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "my_tool" || info.Desc != "my description" {
		t.Fatalf("got %+v, want name=my_tool desc=my description", info)
	}
}

func TestMCPToolInvokableRun(t *testing.T) {
	mt := NewMCPTool(&mockCaller{result: "res"}, "test_tool", "")
	result, err := mt.InvokableRun(context.Background(), `{"key":"val"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if result != "res:test_tool" {
		t.Fatalf("got %q, want %q", result, "res:test_tool")
	}
}

func TestMCPToolEmptyArgs(t *testing.T) {
	mt := NewMCPTool(&mockCaller{result: "ok"}, "empty", "")
	result, err := mt.InvokableRun(context.Background(), "")
	if err != nil {
		t.Fatalf("InvokableRun(empty) error = %v", err)
	}
	if result != "ok:empty" {
		t.Fatalf("got %q, want %q", result, "ok:empty")
	}
}
