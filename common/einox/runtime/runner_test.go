package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"zero-service/common/einox/protocol"
	einoxTool "zero-service/common/einox/tool"
)

func TestRunnerGenerateWithStaticModel(t *testing.T) {
	runner, err := NewRunner(StaticChatModel{Response: "fake-agent-response"})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{
		SessionID: "session-test-001",
		TurnID:    "turn-test-001",
		System:    "You are an Eino helper.",
		Input:     "What is Eino?",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if events[len(events)-1].SessionID != "session-test-001" {
		t.Fatalf("SessionID = %q, want session-test-001", events[len(events)-1].SessionID)
	}
}

func TestRunnerStreamClosesAndCombinesChunks(t *testing.T) {
	runner, err := NewRunner(StaticChatModel{Chunks: []string{"fake-", "agent-", "response"}})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Stream(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageDelta,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	end, ok := decodeData[protocol.MessageEndData](t, events[len(events)-2])
	if !ok || end.Text != "fake-agent-response" {
		t.Fatalf("MessageEnd text = %q, want fake-agent-response", end.Text)
	}
}

func TestRunnerEmitsErrorEvent(t *testing.T) {
	want := errors.New("model unavailable")
	runner, err := NewRunner(StaticChatModel{Err: want})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if !errors.Is(err, want) {
		t.Fatalf("Generate() error = %v, want %v", err, want)
	}
	assertEventTypes(t, events, protocol.EventTurnStart, protocol.EventError, protocol.EventTurnEnd)
}

func TestRunnerGenerateInjectsRAGContext(t *testing.T) {
	calls := &ModelCalls{}
	runner, err := NewRunner(StaticChatModel{Response: "rag-answer", Calls: calls})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{
		SessionID: "session-test-001",
		System:    "You answer with citations.",
		Input:     "What is Eino?",
		RAG: RAGRequest{Retriever: StaticRetriever{Documents: []*schema.Document{
			{ID: "doc-1", Content: "Eino is a Go framework for LLM applications."},
		}}, TopK: 1},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if len(calls.GenerateInput) == 0 {
		t.Fatal("model received no messages")
	}
	system := calls.GenerateInput[0].Content
	if !strings.Contains(system, "You answer with citations.") {
		t.Fatalf("system prompt missing original instructions: %q", system)
	}
	if !strings.Contains(system, "[1] Eino is a Go framework") {
		t.Fatalf("system prompt missing retrieved context: %q", system)
	}
}

func TestRunnerRAGErrorStopsBeforeModel(t *testing.T) {
	want := errors.New("retriever unavailable")
	calls := &ModelCalls{}
	runner, err := NewRunner(StaticChatModel{Response: "unused", Calls: calls})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{
		SessionID: "session-test-001",
		Input:     "What is Eino?",
		RAG:       RAGRequest{Retriever: StaticRetriever{Err: want}},
	})
	if !errors.Is(err, want) {
		t.Fatalf("Generate() error = %v, want %v", err, want)
	}
	assertEventTypes(t, events, protocol.EventTurnStart, protocol.EventError, protocol.EventTurnEnd)
	if len(calls.GenerateInput) != 0 {
		t.Fatalf("model was called after RAG error: %#v", calls.GenerateInput)
	}
}

func TestRunnerStreamRAGErrorStopsBeforeModel(t *testing.T) {
	want := errors.New("retriever unavailable")
	calls := &ModelCalls{}
	runner, err := NewRunner(StaticChatModel{Chunks: []string{"unused"}, Calls: calls})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Stream(context.Background(), Request{
		SessionID: "session-test-001",
		Input:     "What is Eino?",
		RAG:       RAGRequest{Retriever: StaticRetriever{Err: want}},
	})
	if !errors.Is(err, want) {
		t.Fatalf("Stream() error = %v, want %v", err, want)
	}
	assertEventTypes(t, events, protocol.EventTurnStart, protocol.EventError, protocol.EventTurnEnd)
	if len(calls.StreamInput) != 0 {
		t.Fatalf("model was called after RAG error: %#v", calls.StreamInput)
	}
}

func TestRunnerEmptyRAGResultKeepsSystemPromptUnchanged(t *testing.T) {
	calls := &ModelCalls{}
	runner, err := NewRunner(StaticChatModel{Response: "answer", Calls: calls})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Generate(context.Background(), Request{
		SessionID: "session-test-001",
		System:    "base system",
		Input:     "What is Eino?",
		RAG:       RAGRequest{Retriever: StaticRetriever{}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(calls.GenerateInput) == 0 || calls.GenerateInput[0].Content != "base system" {
		t.Fatalf("model system prompt = %#v, want unchanged base system", calls.GenerateInput)
	}
}

func TestRunnerGenerateUsesDefaultRAGRetriever(t *testing.T) {
	calls := &ModelCalls{}
	retriever := &capturingRetriever{docs: []*schema.Document{{ID: "doc-default", Content: "default runtime context"}}}
	runner, err := NewRunner(StaticChatModel{Response: "rag-answer", Calls: calls}, WithRetriever(retriever, 7))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "What is the default context?"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if retriever.query != "What is the default context?" || retriever.topK != 7 {
		t.Fatalf("retriever call = (%q, %d), want request query and default topK 7", retriever.query, retriever.topK)
	}
	if len(calls.GenerateInput) == 0 || !strings.Contains(calls.GenerateInput[0].Content, "default runtime context") {
		t.Fatalf("model system prompt = %#v, want default RAG context", calls.GenerateInput)
	}
}

func TestRunnerGenerateRequestRAGOverridesDefaultRetriever(t *testing.T) {
	calls := &ModelCalls{}
	defaultRetriever := &capturingRetriever{docs: []*schema.Document{{ID: "doc-default", Content: "default context"}}}
	requestRetriever := &capturingRetriever{docs: []*schema.Document{{ID: "doc-request", Content: "request context"}}}
	runner, err := NewRunner(StaticChatModel{Response: "rag-answer", Calls: calls}, WithRetriever(defaultRetriever, 7))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Generate(context.Background(), Request{
		SessionID: "session-test-001",
		Input:     "What is the request context?",
		RAG:       RAGRequest{Retriever: requestRetriever, TopK: 3},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if defaultRetriever.query != "" {
		t.Fatalf("default retriever was called with %q", defaultRetriever.query)
	}
	if requestRetriever.topK != 3 {
		t.Fatalf("request retriever topK = %d, want 3", requestRetriever.topK)
	}
	system := calls.GenerateInput[0].Content
	if !strings.Contains(system, "request context") || strings.Contains(system, "default context") {
		t.Fatalf("system prompt = %q, want request context only", system)
	}
}

func TestKnowledgeRetrieverNoopsWithoutTurnContext(t *testing.T) {
	docs, err := (KnowledgeRetriever{}).Retrieve(context.Background(), "What is Eino?", 5)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("docs = %#v, want empty docs without turn context", docs)
	}
}

func TestRunnerGenerateExecutesToolCallsBeforeFinalAnswer(t *testing.T) {
	modelCalls := &ModelCalls{}
	toolCalls := &ToolCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "tool-result", Calls: toolCalls})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{
		Calls: modelCalls,
		Responses: []*schema.Message{
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"q":"eino"}`}}}),
			schema.AssistantMessage("final answer", nil),
		},
	}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if len(toolCalls.Args) != 1 || toolCalls.Args[0] != `{"q":"eino"}` {
		t.Fatalf("tool args = %#v, want model tool call args", toolCalls.Args)
	}
	start, ok := decodeData[protocol.ToolCallStartData](t, events[1])
	if !ok || start.CallID != "call-1" {
		t.Fatalf("ToolCallStart.CallID = %#v, want call-1", start)
	}
	end, ok := decodeData[protocol.ToolCallEndData](t, events[2])
	if !ok || end.CallID != "call-1" {
		t.Fatalf("ToolCallEnd.CallID = %#v, want call-1", end)
	}
	if len(modelCalls.GenerateInput) == 0 {
		t.Fatal("model received no final-call messages")
	}
	last := modelCalls.GenerateInput[len(modelCalls.GenerateInput)-1]
	if last.Role != schema.Tool || last.Content != "tool-result" || last.ToolCallID != "call-1" || last.ToolName != "lookup" {
		t.Fatalf("last final-call message = %#v, want tool result", last)
	}
}

func TestRunnerGeneratePassesToolInfosToModel(t *testing.T) {
	modelCalls := &ModelCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Desc: "Lookup facts", Result: "tool-result"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{Response: "final answer", Calls: modelCalls}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(modelCalls.GenerateTools) != 1 || modelCalls.GenerateTools[0].Name != "lookup" {
		t.Fatalf("GenerateTools = %#v, want lookup tool info", modelCalls.GenerateTools)
	}
}

func TestRunnerStreamPassesToolInfosToModel(t *testing.T) {
	modelCalls := &ModelCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Desc: "Lookup facts", Result: "tool-result"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{Response: "final answer", Calls: modelCalls}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Stream(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(modelCalls.StreamTools) != 1 || modelCalls.StreamTools[0].Name != "lookup" {
		t.Fatalf("StreamTools = %#v, want lookup tool info", modelCalls.StreamTools)
	}
}

func TestRunnerGenerateExecutesMultipleToolIterations(t *testing.T) {
	modelCalls := &ModelCalls{}
	toolCalls := &ToolCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "tool-result", Calls: toolCalls})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{
		Calls: modelCalls,
		Responses: []*schema.Message{
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"step":1}`}}}),
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-2", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"step":2}`}}}),
			schema.AssistantMessage("final answer", nil),
		},
	}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "multi lookup"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if len(toolCalls.Args) != 2 || toolCalls.Args[0] != `{"step":1}` || toolCalls.Args[1] != `{"step":2}` {
		t.Fatalf("tool args = %#v, want two tool iterations", toolCalls.Args)
	}
	last := modelCalls.GenerateInput[len(modelCalls.GenerateInput)-1]
	if last.Role != schema.Tool || last.ToolCallID != "call-2" {
		t.Fatalf("last final-call message = %#v, want second tool result", last)
	}
}

func TestRunnerGenerateStopsAtMaxToolIterations(t *testing.T) {
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "tool-result"})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{
		Responses: []*schema.Message{
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{}`}}}),
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-2", Function: schema.FunctionCall{Name: "lookup", Arguments: `{}`}}}),
		},
	}, WithTools(tools), WithMaxToolIterations(1))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "loop"})
	if err == nil {
		t.Fatal("Generate() error = nil, want max tool iteration error")
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventError,
		protocol.EventTurnEnd,
	)
}

func TestRunnerGenerateReturnsToolCallError(t *testing.T) {
	runner, err := NewRunner(StaticChatModel{Responses: []*schema.Message{
		schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "missing", Arguments: `{}`}}}),
	}}, WithTools(&ToolRegistry{}))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "call missing"})
	if err == nil {
		t.Fatal("Generate() error = nil, want tool lookup error")
	}
	assertEventTypes(t, events, protocol.EventTurnStart, protocol.EventToolCallStart, protocol.EventToolCallEnd, protocol.EventError, protocol.EventTurnEnd)
}

func TestRunnerGenerateRequestToolsOverrideDefaultTools(t *testing.T) {
	defaultTools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "default-result"})
	if err != nil {
		t.Fatalf("NewToolRegistry(default) error = %v", err)
	}
	requestTools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "request-result"})
	if err != nil {
		t.Fatalf("NewToolRegistry(request) error = %v", err)
	}
	modelCalls := &ModelCalls{}
	runner, err := NewRunner(StaticChatModel{
		Calls: modelCalls,
		Responses: []*schema.Message{
			schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{}`}}}),
			schema.AssistantMessage("final answer", nil),
		},
	}, WithTools(defaultTools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	_, err = runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "lookup", Tools: requestTools})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	last := modelCalls.GenerateInput[len(modelCalls.GenerateInput)-1]
	if last.Role != schema.Tool || last.Content != "request-result" {
		t.Fatalf("last final-call message = %#v, want request tool result", last)
	}
}

func TestRunnerStreamExecutesToolCallsBeforeFinalAnswer(t *testing.T) {
	modelCalls := &ModelCalls{}
	toolCalls := &ToolCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "stream-tool-result", Calls: toolCalls})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{
		Calls: modelCalls,
		Streams: [][]*schema.Message{
			{schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"q":"stream"}`}}})},
			{schema.AssistantMessage("stream ", nil), schema.AssistantMessage("answer", nil)},
		},
	}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Stream(context.Background(), Request{SessionID: "session-test-001", Input: "What is Eino?"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if len(toolCalls.Args) != 1 || toolCalls.Args[0] != `{"q":"stream"}` {
		t.Fatalf("tool args = %#v, want stream tool call args", toolCalls.Args)
	}
	start, ok := decodeData[protocol.ToolCallStartData](t, events[1])
	if !ok || start.CallID != "call-1" {
		t.Fatalf("ToolCallStart.CallID = %#v, want call-1", start)
	}
	end, ok := decodeData[protocol.ToolCallEndData](t, events[2])
	if !ok || end.CallID != "call-1" {
		t.Fatalf("ToolCallEnd.CallID = %#v, want call-1", end)
	}
	if len(modelCalls.StreamInput) == 0 {
		t.Fatal("model received no final stream messages")
	}
	last := modelCalls.StreamInput[len(modelCalls.StreamInput)-1]
	if last.Role != schema.Tool || last.Content != "stream-tool-result" || last.ToolCallID != "call-1" || last.ToolName != "lookup" {
		t.Fatalf("last final stream message = %#v, want tool result", last)
	}
	turnEnd, ok := decodeData[protocol.TurnEndData](t, events[len(events)-1])
	if !ok || turnEnd.LastMessage != "stream answer" {
		t.Fatalf("turn end = %#v, want final streamed answer", turnEnd)
	}
}

func TestRunnerStreamExecutesMultipleToolIterations(t *testing.T) {
	modelCalls := &ModelCalls{}
	toolCalls := &ToolCalls{}
	tools, err := NewToolRegistry(StaticTool{Name: "lookup", Result: "stream-tool-result", Calls: toolCalls})
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{
		Calls: modelCalls,
		Streams: [][]*schema.Message{
			{schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"step":1}`}}})},
			{schema.AssistantMessage("", []schema.ToolCall{{ID: "call-2", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"step":2}`}}})},
			{schema.AssistantMessage("stream final", nil)},
		},
	}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Stream(context.Background(), Request{SessionID: "session-test-001", Input: "multi stream lookup"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	if len(toolCalls.Args) != 2 || toolCalls.Args[0] != `{"step":1}` || toolCalls.Args[1] != `{"step":2}` {
		t.Fatalf("tool args = %#v, want two stream tool iterations", toolCalls.Args)
	}
	last := modelCalls.StreamInput[len(modelCalls.StreamInput)-1]
	if last.Role != schema.Tool || last.ToolCallID != "call-2" {
		t.Fatalf("last final stream message = %#v, want second tool result", last)
	}
}

func TestRunnerStreamReturnsToolCallError(t *testing.T) {
	runner, err := NewRunner(StaticChatModel{Streams: [][]*schema.Message{
		{schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "missing", Arguments: `{}`}}})},
	}}, WithTools(&ToolRegistry{}))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Stream(context.Background(), Request{SessionID: "session-test-001", Input: "call missing"})
	if err == nil {
		t.Fatal("Stream() error = nil, want tool lookup error")
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventError,
		protocol.EventTurnEnd,
	)
}

func TestRunnerGenerateCannotBypassPolicyFilteredRegistry(t *testing.T) {
	allowedCalls := &ToolCalls{}
	blockedCalls := &ToolCalls{}
	kit := einoxTool.NewKit()
	if err := kit.Register(einoxTool.CapCompute, StaticTool{Name: "allowed", Result: "allowed-result", Calls: allowedCalls}); err != nil {
		t.Fatalf("register allowed tool: %v", err)
	}
	if err := kit.Register(einoxTool.CapIO, StaticTool{Name: "blocked", Result: "blocked-result", Calls: blockedCalls}); err != nil {
		t.Fatalf("register blocked tool: %v", err)
	}

	tools, err := NewToolRegistry(einoxTool.NewPolicy().AllowCapabilities(einoxTool.CapCompute).Apply(kit)...)
	if err != nil {
		t.Fatalf("NewToolRegistry() error = %v", err)
	}
	runner, err := NewRunner(StaticChatModel{Responses: []*schema.Message{
		schema.AssistantMessage("", []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "blocked", Arguments: `{}`}}}),
	}}, WithTools(tools))
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	events, err := runner.Generate(context.Background(), Request{SessionID: "session-test-001", Input: "call blocked"})
	if err == nil || !strings.Contains(err.Error(), `tool "blocked" not registered`) {
		t.Fatalf("Generate() error = %v, want blocked tool lookup error", err)
	}
	if len(blockedCalls.Args) != 0 {
		t.Fatalf("blocked tool was invoked: %#v", blockedCalls.Args)
	}
	if len(allowedCalls.Args) != 0 {
		t.Fatalf("allowed tool unexpectedly invoked: %#v", allowedCalls.Args)
	}
	assertEventTypes(t, events,
		protocol.EventTurnStart,
		protocol.EventToolCallStart,
		protocol.EventToolCallEnd,
		protocol.EventError,
		protocol.EventTurnEnd,
	)
}

func TestRunnerLiteRuntimeHasNoResumeSurface(t *testing.T) {
	runner, err := NewRunner(StaticChatModel{Response: "ok"})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	type adkResumeSurface interface {
		Resume(context.Context, string)
	}
	if _, ok := any(runner).(adkResumeSurface); ok {
		t.Fatal("runtime Runner exposes a Resume surface; lite runtime should remain non-resumable")
	}
}

func TestMessagesAndLastText(t *testing.T) {
	history := []*schema.Message{schema.UserMessage("old question"), schema.AssistantMessage("old answer", nil)}
	msgs := Messages("system", history, "new question")
	if len(msgs) != 4 {
		t.Fatalf("len(Messages) = %d, want 4", len(msgs))
	}
	if LastText(msgs) != "new question" {
		t.Fatalf("LastText() = %q, want new question", LastText(msgs))
	}
}

func assertEventTypes(t *testing.T, events []protocol.Event, want ...protocol.EventType) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("event count = %d, want %d: %#v", len(events), len(want), events)
	}
	for i, event := range events {
		if event.Type != want[i] {
			t.Fatalf("event[%d].Type = %q, want %q", i, event.Type, want[i])
		}
	}
}

type capturingRetriever struct {
	docs  []*schema.Document
	err   error
	query string
	topK  int
}

func (r *capturingRetriever) Retrieve(_ context.Context, query string, topK int) ([]*schema.Document, error) {
	r.query = query
	r.topK = topK
	if r.err != nil {
		return nil, r.err
	}
	return r.docs, nil
}
