package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"zero-service/common/einox/protocol"
)

type Runner struct {
	model             model.BaseChatModel
	tools             *ToolRegistry
	retriever         Retriever
	ragTopK           int
	maxToolIterations int
}

const defaultMaxToolIterations = 8

type RunnerOption func(*Runner)

type Request struct {
	SessionID string
	TurnID    string
	System    string
	History   []*schema.Message
	Input     string
	RAG       RAGRequest
	Tools     *ToolRegistry
}

type RAGRequest struct {
	Retriever Retriever
	TopK      int
}

func NewRunner(chatModel model.BaseChatModel, opts ...RunnerOption) (*Runner, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("runtime: chat model is nil")
	}
	r := &Runner{model: chatModel}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

func WithTools(tools *ToolRegistry) RunnerOption {
	return func(r *Runner) {
		r.tools = tools
	}
}

func WithRetriever(retriever Retriever, topK int) RunnerOption {
	return func(r *Runner) {
		r.retriever = retriever
		r.ragTopK = topK
	}
}

func WithMaxToolIterations(max int) RunnerOption {
	return func(r *Runner) {
		r.maxToolIterations = max
	}
}

func (r *Runner) Generate(ctx context.Context, req Request) ([]protocol.Event, error) {
	collector := newEventCollector(req)
	em := protocol.NewEmitter(collector, collector.sessionID, collector.turnID)
	_ = em.TurnStart(protocol.TurnStartData{UserMessage: strings.TrimSpace(req.Input)})

	msgs, err := r.messages(ctx, req)
	if err != nil {
		_ = em.EmitError("rag_retrieve", err.Error())
		_ = em.TurnEnd(false, "", "")
		return collector.events, err
	}

	modelOpts, err := r.modelOptions(ctx, req)
	if err != nil {
		_ = em.EmitError("tool_info", err.Error())
		_ = em.TurnEnd(false, "", "")
		return collector.events, err
	}

	msg, err := r.model.Generate(ctx, msgs, modelOpts...)
	if err != nil {
		_ = em.EmitError("model_generate", err.Error())
		_ = em.TurnEnd(false, "", "")
		return collector.events, err
	}
	toolIterations := 0
	for msg != nil && len(msg.ToolCalls) > 0 {
		if toolIterations >= r.effectiveMaxToolIterations() {
			err := fmt.Errorf("runtime: max tool iterations exceeded")
			_ = em.EmitError("tool_call", err.Error())
			_ = em.TurnEnd(false, "", "")
			return collector.events, err
		}
		toolMsgs, err := r.runToolCalls(ctx, em, req, msg)
		if err != nil {
			_ = em.EmitError("tool_call", err.Error())
			_ = em.TurnEnd(false, "", "")
			return collector.events, err
		}
		msgs = append(msgs, msg)
		msgs = append(msgs, toolMsgs...)
		toolIterations++
		msg, err = r.model.Generate(ctx, msgs, modelOpts...)
		if err != nil {
			_ = em.EmitError("model_generate", err.Error())
			_ = em.TurnEnd(false, "", "")
			return collector.events, err
		}
	}
	text := ""
	if msg != nil {
		text = msg.Content
	}
	messageID := uuid.NewString()
	_ = em.Emit(protocol.EventMessageStart, protocol.MessageStartData{MessageID: messageID, Role: protocol.RoleAssistant})
	if text != "" {
		_ = em.Emit(protocol.EventMessageDelta, protocol.MessageDeltaData{MessageID: messageID, Text: text})
	}
	_ = em.Emit(protocol.EventMessageEnd, protocol.MessageEndData{MessageID: messageID, Role: protocol.RoleAssistant, Text: text})
	_ = em.TurnEnd(false, "", text)
	return collector.events, nil
}

func (r *Runner) Stream(ctx context.Context, req Request) ([]protocol.Event, error) {
	collector := newEventCollector(req)
	em := protocol.NewEmitter(collector, collector.sessionID, collector.turnID)
	_ = em.TurnStart(protocol.TurnStartData{UserMessage: strings.TrimSpace(req.Input)})

	msgs, err := r.messages(ctx, req)
	if err != nil {
		_ = em.EmitError("rag_retrieve", err.Error())
		_ = em.TurnEnd(false, "", "")
		return collector.events, err
	}

	modelOpts, err := r.modelOptions(ctx, req)
	if err != nil {
		_ = em.EmitError("tool_info", err.Error())
		_ = em.TurnEnd(false, "", "")
		return collector.events, err
	}

	msg, text, err := r.streamModel(ctx, em, msgs, modelOpts...)
	if err != nil {
		return collector.events, err
	}
	toolIterations := 0
	for msg != nil && len(msg.ToolCalls) > 0 {
		if toolIterations >= r.effectiveMaxToolIterations() {
			err := fmt.Errorf("runtime: max tool iterations exceeded")
			_ = em.EmitError("tool_call", err.Error())
			_ = em.TurnEnd(false, "", text)
			return collector.events, err
		}
		toolMsgs, err := r.runToolCalls(ctx, em, req, msg)
		if err != nil {
			_ = em.EmitError("tool_call", err.Error())
			_ = em.TurnEnd(false, "", "")
			return collector.events, err
		}
		msgs = append(msgs, msg)
		msgs = append(msgs, toolMsgs...)
		toolIterations++
		msg, text, err = r.streamModel(ctx, em, msgs, modelOpts...)
		if err != nil {
			return collector.events, err
		}
	}
	_ = em.TurnEnd(false, "", text)
	return collector.events, nil
}

func (r *Runner) streamModel(ctx context.Context, em *protocol.Emitter, msgs []*schema.Message, opts ...model.Option) (*schema.Message, string, error) {
	stream, err := r.model.Stream(ctx, msgs, opts...)
	if err != nil {
		_ = em.EmitError("model_stream", err.Error())
		_ = em.TurnEnd(false, "", "")
		return nil, "", err
	}
	defer stream.Close()

	messageID := uuid.NewString()
	var full strings.Builder
	chunks := make([]*schema.Message, 0)
	started := false
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			_ = em.EmitError("model_stream_recv", recvErr.Error())
			_ = em.TurnEnd(false, "", full.String())
			return nil, full.String(), recvErr
		}
		if chunk == nil {
			continue
		}
		chunks = append(chunks, chunk)
		if chunk.Content == "" {
			continue
		}
		if !started {
			_ = em.Emit(protocol.EventMessageStart, protocol.MessageStartData{MessageID: messageID, Role: protocol.RoleAssistant})
			started = true
		}
		full.WriteString(chunk.Content)
		_ = em.Emit(protocol.EventMessageDelta, protocol.MessageDeltaData{MessageID: messageID, Text: chunk.Content})
	}
	if !started {
		msg, err := schema.ConcatMessages(chunks)
		if err != nil {
			_ = em.EmitError("model_stream_concat", err.Error())
			return nil, "", err
		}
		return msg, "", nil
	}
	text := full.String()
	_ = em.Emit(protocol.EventMessageEnd, protocol.MessageEndData{MessageID: messageID, Role: protocol.RoleAssistant, Text: text})
	msg, err := schema.ConcatMessages(chunks)
	if err != nil {
		_ = em.EmitError("model_stream_concat", err.Error())
		return nil, text, err
	}
	return msg, text, nil
}

func (r *Runner) runToolCalls(ctx context.Context, em *protocol.Emitter, req Request, msg *schema.Message) ([]*schema.Message, error) {
	tools := r.effectiveTools(req)
	if tools == nil {
		return nil, fmt.Errorf("runtime: model requested tools but no tool registry is configured")
	}
	toolMsgs := make([]*schema.Message, 0, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			return nil, fmt.Errorf("runtime: tool call %q has empty function name", call.ID)
		}
		result, err := tools.RunWithEmitterCallID(ctx, em, call.ID, name, call.Function.Arguments)
		if err != nil {
			return nil, err
		}
		toolMsgs = append(toolMsgs, schema.ToolMessage(result, call.ID, schema.WithToolName(name)))
	}
	return toolMsgs, nil
}

func (r *Runner) modelOptions(ctx context.Context, req Request) ([]model.Option, error) {
	tools := r.effectiveTools(req)
	if tools == nil {
		return nil, nil
	}
	infos, err := tools.Infos(ctx)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, nil
	}
	return []model.Option{model.WithTools(infos)}, nil
}

func (r *Runner) effectiveTools(req Request) *ToolRegistry {
	if req.Tools != nil {
		return req.Tools
	}
	return r.tools
}

func (r *Runner) effectiveMaxToolIterations() int {
	if r.maxToolIterations <= 0 {
		return defaultMaxToolIterations
	}
	return r.maxToolIterations
}

func (r *Runner) messages(ctx context.Context, req Request) ([]*schema.Message, error) {
	system := req.System
	retriever := req.RAG.Retriever
	if retriever == nil {
		retriever = r.retriever
	}
	if retriever != nil {
		topK := req.RAG.TopK
		if topK <= 0 {
			topK = r.ragTopK
		}
		docs, err := retriever.Retrieve(ctx, req.Input, topK)
		if err != nil {
			return nil, err
		}
		if contextBlock := DocumentsContext(docs); contextBlock != "" {
			system = AppendSystemContext(system, contextBlock)
		}
	}
	return Messages(system, req.History, req.Input), nil
}

type eventCollector struct {
	sessionID string
	turnID    string
	events    []protocol.Event
}

func newEventCollector(req Request) *eventCollector {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "session-test-001"
	}
	turnID := strings.TrimSpace(req.TurnID)
	if turnID == "" {
		turnID = uuid.NewString()
	}
	return &eventCollector{sessionID: sessionID, turnID: turnID}
}

func (c *eventCollector) Write(p []byte) (int, error) {
	event, err := protocol.Decode(p)
	if err != nil {
		return 0, err
	}
	c.events = append(c.events, event)
	return len(p), nil
}
