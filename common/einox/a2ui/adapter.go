package a2ui

import (
	"encoding/json"
	"io"

	"zero-service/common/stream"
)

// =============================================================================
// StreamEvent 适配器
// 将 common/stream.StreamEvent 转换为 A2UI 消息
// =============================================================================

// StreamEventToWriter 将 stream.StreamEvent 流转换为 A2UI JSON 并写入 w
func StreamEventToWriter(w io.Writer, events <-chan *stream.StreamEvent) error {
	for event := range events {
		if err := writeStreamEvent(w, event); err != nil {
			return err
		}
	}
	return nil
}

// writeStreamEvent 将单个 StreamEvent 转换为 A2UI 消息
func writeStreamEvent(w io.Writer, event *stream.StreamEvent) error {
	switch event.Type {
	case stream.EventText:
		return writeTextEvent(w, event)
	case stream.EventToolCall:
		return writeToolCallEvent(w, event)
	case stream.EventToolResult:
		return writeToolResultEvent(w, event)
	case stream.EventThinking:
		return writeThinkingEvent(w, event)
	case stream.EventInterrupt:
		return writeInterruptEvent(w, event)
	case stream.EventError:
		return writeErrorEvent(w, event)
	case stream.EventDone:
		return writeDoneEvent(w)
	default:
		return nil
	}
}

// writeTextEvent 写入文本事件
func writeTextEvent(w io.Writer, event *stream.StreamEvent) error {
	content, ok := event.Data.(string)
	if !ok {
		return nil
	}
	msg := NewTextEvent(content, false)
	return emitA2UIMessage(w, msg)
}

// writeToolCallEvent 写入工具调用事件
func writeToolCallEvent(w io.Writer, event *stream.StreamEvent) error {
	data, ok := event.Data.(map[string]string)
	if !ok {
		return nil
	}
	name := data["name"]
	args := data["args"]
	msg, err := NewToolCallEvent(name, name, args)
	if err != nil {
		return err
	}
	return emitA2UIMessage(w, msg)
}

// writeToolResultEvent 写入工具结果事件
func writeToolResultEvent(w io.Writer, event *stream.StreamEvent) error {
	data, ok := event.Data.(map[string]string)
	if !ok {
		return nil
	}
	name := data["name"]
	result := data["result"]
	msg := NewToolResultEvent(name, name, result, "")
	return emitA2UIMessage(w, msg)
}

// writeThinkingEvent 写入思考事件
func writeThinkingEvent(w io.Writer, event *stream.StreamEvent) error {
	content, ok := event.Data.(string)
	if !ok {
		return nil
	}
	msg := NewThinkingEvent(content, false)
	return emitA2UIMessage(w, msg)
}

// writeInterruptEvent 写入中断事件
func writeInterruptEvent(w io.Writer, event *stream.StreamEvent) error {
	msg := &Event{
		Type:      EventToolCall,
		Component: ComponentTypeToolCall,
		Data:      event.Data,
	}
	return emitA2UIMessage(w, msg)
}

// writeErrorEvent 写入错误事件
func writeErrorEvent(w io.Writer, event *stream.StreamEvent) error {
	content, ok := event.Data.(string)
	if !ok {
		return nil
	}
	msg := NewErrorEvent(content)
	return emitA2UIMessage(w, msg)
}

// writeDoneEvent 写入完成事件
func writeDoneEvent(w io.Writer) error {
	msg := NewEndEvent()
	return emitA2UIMessage(w, msg)
}

// emitA2UIMessage 将 A2UI Event 编码并写入 io.Writer
func emitA2UIMessage(w io.Writer, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// =============================================================================
// WriterToStream 适配器
// 将 io.Writer 适配为 stream.Sender
// =============================================================================

// WriterAdapter 将 io.Writer 适配为 stream.Sender
type WriterAdapter struct {
	w       io.Writer
	session string
}

// NewWriterAdapter 创建 Writer 适配器
func NewWriterAdapter(w io.Writer, sessionID string) *WriterAdapter {
	return &WriterAdapter{w: w, session: sessionID}
}

// Write 实现 io.Writer
func (a *WriterAdapter) Write(p []byte) (n int, err error) {
	return a.w.Write(p)
}

// SendJSON 发送 JSON 消息
func (a *WriterAdapter) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = a.w.Write(data)
	return err
}

// SendDone 发送完成信号
func (a *WriterAdapter) SendDone() {
	// A2UI 完成信号
	event := NewEndEvent()
	_ = emitA2UIMessage(a.w, event)
}

// SendError 发送错误信号
func (a *WriterAdapter) SendError(err error) {
	event := NewErrorEvent(err.Error())
	_ = emitA2UIMessage(a.w, event)
}

// SessionID 返回会话 ID
func (a *WriterAdapter) SessionID() string {
	return a.session
}

// Ensure WriterAdapter implements stream.Sender
var _ stream.Sender = (*WriterAdapter)(nil)

// Ensure WriterAdapter implements io.Writer
var _ io.Writer = (*WriterAdapter)(nil)
