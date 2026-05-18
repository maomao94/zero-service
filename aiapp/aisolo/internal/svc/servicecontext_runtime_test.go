package svc

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"

	einoxruntime "zero-service/common/einox/runtime"
	"zero-service/common/einox/tool/builtin"
)

func TestInitRuntimeRunnerWithStaticModel(t *testing.T) {
	s := &ServiceContext{ChatModel: einoxruntime.StaticChatModel{Response: "runtime-ok"}}

	s.initRuntimeRunner()

	if s.RuntimeRunner == nil {
		t.Fatal("RuntimeRunner is nil")
	}
	events, err := s.RuntimeRunner.Generate(context.Background(), einoxruntime.Request{
		SessionID: "session-test-001",
		TurnID:    "turn-test-001",
		Input:     "ping",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Generate() returned no events")
	}
}

func TestInitRuntimeRunnerSkipsNilModel(t *testing.T) {
	s := &ServiceContext{}

	s.initRuntimeRunner()

	if s.RuntimeRunner != nil {
		t.Fatal("RuntimeRunner should stay nil when ChatModel is nil")
	}
}

func TestInitRuntimeRunnerUsesDefaultRuntimeTools(t *testing.T) {
	modelCalls := &einoxruntime.ModelCalls{}
	s := &ServiceContext{
		ChatModel: einoxruntime.StaticChatModel{
			Calls: modelCalls,
			Responses: []*schema.Message{
				schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "echo", Arguments: `{"text":"hello"}`}}}),
				schema.AssistantMessage("final answer", nil),
			},
		},
		Kit: builtin.MustNewDefaultKit(),
	}

	s.initRuntimeTools(context.Background())
	s.initRuntimeRunner()

	if s.RuntimeRunner == nil {
		t.Fatal("RuntimeRunner is nil")
	}
	_, err := s.RuntimeRunner.Generate(context.Background(), einoxruntime.Request{SessionID: "session-test-001", Input: "echo"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	last := modelCalls.GenerateInput[len(modelCalls.GenerateInput)-1]
	if last.Role != schema.Tool || last.ToolName != "echo" || last.Content == "" {
		t.Fatalf("last final-call message = %#v, want default echo tool result", last)
	}
}

func TestInitRuntimeToolsFromKit(t *testing.T) {
	s := &ServiceContext{Kit: builtin.MustNewDefaultKit()}

	s.initRuntimeTools(context.Background())

	if s.RuntimeTools == nil {
		t.Fatal("RuntimeTools is nil")
	}
	infos, err := s.RuntimeTools.Infos(context.Background())
	if err != nil {
		t.Fatalf("Infos() error = %v", err)
	}
	if len(infos) == 0 {
		t.Fatal("RuntimeTools has no tool info")
	}
	assertRuntimeToolNames(t, infos, []string{"echo", "calculator", "now", "random_id"})
	assertRuntimeToolAbsent(t, infos, "ask_confirm")
	result, err := s.RuntimeTools.Run(context.Background(), "echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("Run(echo) error = %v", err)
	}
	if result == "" {
		t.Fatal("Run(echo) returned empty result")
	}
	if _, err := s.RuntimeTools.Run(context.Background(), "ask_confirm", `{"question":"continue?"}`); err == nil {
		t.Fatal("Run(ask_confirm) succeeded; runtime tools should not include human interrupt tools")
	}
}

func TestInitRuntimeToolsSkipsNilKit(t *testing.T) {
	s := &ServiceContext{}

	s.initRuntimeTools(context.Background())

	if s.RuntimeTools != nil {
		t.Fatal("RuntimeTools should stay nil when Kit is nil")
	}
}

func assertRuntimeToolNames(t *testing.T, infos []*schema.ToolInfo, names []string) {
	t.Helper()
	found := make(map[string]struct{}, len(infos))
	for _, info := range infos {
		if info != nil {
			found[info.Name] = struct{}{}
		}
	}
	for _, name := range names {
		if _, ok := found[name]; !ok {
			t.Fatalf("runtime tool %q missing from infos %#v", name, found)
		}
	}
}

func assertRuntimeToolAbsent(t *testing.T, infos []*schema.ToolInfo, name string) {
	t.Helper()
	for _, info := range infos {
		if info != nil && info.Name == name {
			t.Fatalf("runtime tool %q should not be registered", name)
		}
	}
}
