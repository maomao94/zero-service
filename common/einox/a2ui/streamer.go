package a2ui

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// EventHandler 事件处理器
type EventHandler func(*Event) error

// =============================================================================
// Streamer
// =============================================================================

// Streamer 流式渲染器
type Streamer struct {
	mu       sync.RWMutex
	events   []*Event
	handlers []EventHandler
	writer   *SSEWriter
}

// NewStreamer 创建流式渲染器
func NewStreamer(w io.Writer) *Streamer {
	return &Streamer{
		events:   make([]*Event, 0),
		handlers: make([]EventHandler, 0),
		writer:   NewSSEWriter(w),
	}
}

// NewStreamerWithWriter 使用 SSEWriter 创建
func NewStreamerWithWriter(writer *SSEWriter) *Streamer {
	return &Streamer{
		events:   make([]*Event, 0),
		handlers: make([]EventHandler, 0),
		writer:   writer,
	}
}

// AddHandler 添加事件处理器
func (s *Streamer) AddHandler(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// Emit 发送事件
func (s *Streamer) Emit(event *Event) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.Unlock()

	// 写入 SSE
	if s.writer != nil {
		if err := s.writer.Write(event); err != nil {
			return err
		}
	}

	// 触发处理器
	for _, handler := range handlers {
		handler(event)
	}

	return nil
}

// EmitText 发送文本
func (s *Streamer) EmitText(content string, finish bool) error {
	return s.Emit(NewTextEvent(content, finish))
}

// EmitMarkdown 发送 Markdown
func (s *Streamer) EmitMarkdown(content string, finish bool) error {
	return s.Emit(NewMarkdownEvent(content, finish))
}

// EmitToolCall 发送工具调用
func (s *Streamer) EmitToolCall(id, name string, args any) error {
	event, err := NewToolCallEvent(id, name, args)
	if err != nil {
		return err
	}
	return s.Emit(event)
}

// EmitToolResult 发送工具结果
func (s *Streamer) EmitToolResult(id, name, result, errMsg string) error {
	return s.Emit(NewToolResultEvent(id, name, result, errMsg))
}

// EmitCode 发送代码块
func (s *Streamer) EmitCode(language, code string) error {
	return s.Emit(NewCodeEvent(language, code, false))
}

// EmitError 发送错误
func (s *Streamer) EmitError(errMsg string) error {
	return s.Emit(NewErrorEvent(errMsg))
}

// EmitStart 发送开始
func (s *Streamer) EmitStart() error {
	return s.Emit(NewStartEvent())
}

// EmitEnd 发送结束
func (s *Streamer) EmitEnd() error {
	return s.Emit(NewEndEvent())
}

// GetEvents 获取所有事件
func (s *Streamer) GetEvents() []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Event, len(s.events))
	copy(result, s.events)
	return result
}

// Flush 刷新缓冲区
func (s *Streamer) Flush() error {
	if s.writer != nil {
		return s.writer.Flush()
	}
	return nil
}

// WriteJSON 将事件流写入 JSON 格式
func (s *Streamer) WriteJSON(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return writeEventsJSON(w, s.events)
}

// =============================================================================
// 带超时的流式上下文
// =============================================================================

// StreamContext 带 Streamer 的上下文
type StreamContext struct {
	context.Context
	streamer *Streamer
	cancel   context.CancelFunc
}

// WithTimeout 创建带超时的流式上下文
func WithTimeout(ctx context.Context, timeout time.Duration, w io.Writer) (*StreamContext, *Streamer) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	streamer := NewStreamer(w)
	return &StreamContext{
		Context:  ctx,
		streamer: streamer,
		cancel:   cancel,
	}, streamer
}

func (c *StreamContext) Done() <-chan struct{} {
	return c.Context.Done()
}

func (c *StreamContext) Err() error {
	return c.Context.Err()
}

func (c *StreamContext) Streamer() *Streamer {
	return c.streamer
}

func (c *StreamContext) Cancel() {
	c.cancel()
}

// =============================================================================
// 辅助函数
// =============================================================================

func writeEventsJSON(w io.Writer, events []*Event) error {
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}
