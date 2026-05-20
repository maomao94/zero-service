package protocol

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestPipeEventsReturnsAssistantStreamError(t *testing.T) {
	streamErr := errors.New("stream failed")
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		writer.Send(schema.AssistantMessage("hello", nil), nil)
		writer.Send(nil, streamErr)
		writer.Close()
	}()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(nil, reader, schema.Assistant, ""))
	gen.Close()

	var out bytes.Buffer
	_, err := PipeEvents(NewEmitter(&out, "sess-1", "turn-1"), iter, PipeOptions{})
	if err == nil || !strings.Contains(err.Error(), "stream failed") {
		t.Fatalf("PipeEvents() error = %v, want stream failure", err)
	}

	events := decodeEvents(t, out.Bytes())
	if len(events) == 0 || events[len(events)-1].Type != EventError {
		t.Fatalf("events = %#v, want final error event", events)
	}
}

func TestPipeEventsReturnsToolStreamError(t *testing.T) {
	streamErr := errors.New("tool stream failed")
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		writer.Send(schema.ToolMessage("partial", "call-1"), nil)
		writer.Send(nil, streamErr)
		writer.Close()
	}()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(nil, reader, schema.Tool, "calculator"))
	gen.Close()

	var out bytes.Buffer
	_, err := PipeEvents(NewEmitter(&out, "sess-1", "turn-1"), iter, PipeOptions{})
	if err == nil || !strings.Contains(err.Error(), "tool stream failed") {
		t.Fatalf("PipeEvents() error = %v, want tool stream failure", err)
	}

	events := decodeEvents(t, out.Bytes())
	if len(events) == 0 || events[len(events)-1].Type != EventError {
		t.Fatalf("events = %#v, want final error event", events)
	}
}

func TestPipeEventsEmitsToolCallOnlyStream(t *testing.T) {
	idx := 0
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		writer.Send(&schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{Index: &idx, ID: "call-1", Function: schema.FunctionCall{Name: "calculator", Arguments: `{"expr":"1+1"}`}}}}, nil)
		writer.Close()
	}()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(nil, reader, schema.Assistant, ""))
	gen.Close()

	var out bytes.Buffer
	_, err := PipeEvents(NewEmitter(&out, "sess-1", "turn-1"), iter, PipeOptions{})
	if err != nil {
		t.Fatalf("PipeEvents() error = %v, want nil", err)
	}

	events := decodeEvents(t, out.Bytes())
	for _, event := range events {
		if event.Type == EventToolCallStart {
			return
		}
	}
	t.Fatalf("events = %#v, want tool call start", events)
}

func decodeEvents(t *testing.T, data []byte) []Event {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		event, err := Decode(line)
		if err != nil {
			t.Fatalf("Decode(%q): %v", line, err)
		}
		events = append(events, event)
	}
	return events
}
