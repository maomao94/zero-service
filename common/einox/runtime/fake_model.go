package runtime

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// StaticChatModel is a deterministic local model for SDK tests and examples.
type StaticChatModel struct {
	Response  string
	Responses []*schema.Message
	Chunks    []string
	Streams   [][]*schema.Message
	Err       error
	Calls     *ModelCalls
}

type ModelCalls struct {
	mu              sync.Mutex
	generateCalls   int
	streamCalls     int
	GenerateInput   []*schema.Message
	StreamInput     []*schema.Message
	GenerateTools   []*schema.ToolInfo
	StreamTools     []*schema.ToolInfo
	GenerateOptions []model.Option
	StreamOptions   []model.Option
}

func (m StaticChatModel) Generate(_ context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	callIndex := 0
	if m.Calls != nil {
		options := model.GetCommonOptions(&model.Options{}, opts...)
		m.Calls.mu.Lock()
		callIndex = m.Calls.generateCalls
		m.Calls.generateCalls++
		m.Calls.GenerateInput = append([]*schema.Message(nil), input...)
		m.Calls.GenerateTools = append([]*schema.ToolInfo(nil), options.Tools...)
		m.Calls.GenerateOptions = append([]model.Option(nil), opts...)
		m.Calls.mu.Unlock()
	}
	if len(m.Responses) > 0 {
		if callIndex < len(m.Responses) {
			return m.Responses[callIndex], nil
		}
		return m.Responses[len(m.Responses)-1], nil
	}
	return schema.AssistantMessage(m.Response, nil), nil
}

func (m StaticChatModel) Stream(_ context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.Err != nil {
		return nil, m.Err
	}
	callIndex := 0
	if m.Calls != nil {
		options := model.GetCommonOptions(&model.Options{}, opts...)
		m.Calls.mu.Lock()
		callIndex = m.Calls.streamCalls
		m.Calls.streamCalls++
		m.Calls.StreamInput = append([]*schema.Message(nil), input...)
		m.Calls.StreamTools = append([]*schema.ToolInfo(nil), options.Tools...)
		m.Calls.StreamOptions = append([]model.Option(nil), opts...)
		m.Calls.mu.Unlock()
	}
	if len(m.Streams) > 0 {
		stream := m.Streams[len(m.Streams)-1]
		if callIndex < len(m.Streams) {
			stream = m.Streams[callIndex]
		}
		sr, sw := schema.Pipe[*schema.Message](len(stream))
		go func() {
			defer sw.Close()
			for _, chunk := range stream {
				sw.Send(chunk, nil)
			}
		}()
		return sr, nil
	}
	chunks := m.Chunks
	if len(chunks) == 0 {
		chunks = []string{m.Response}
	}
	sr, sw := schema.Pipe[*schema.Message](len(chunks))
	go func() {
		defer sw.Close()
		for _, chunk := range chunks {
			sw.Send(schema.AssistantMessage(chunk, nil), nil)
		}
	}()
	return sr, nil
}

var _ model.BaseChatModel = StaticChatModel{}
