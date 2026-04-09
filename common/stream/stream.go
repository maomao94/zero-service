package stream

import (
	"encoding/json"
	"io"
)

// =============================================================================
// Sender 接口定义
// =============================================================================

// Sender 流发送器接口，封装流式输出的通用操作
// 支持 SSE、gRPC、WebSocket 等多种流式协议
type Sender interface {
	// Write 写入原始字节数据
	io.Writer

	// SendJSON 发送 JSON 编码的消息
	SendJSON(v any) error

	// SendDone 发送流结束信号
	SendDone()

	// SendError 发送错误信号
	SendError(err error)
}

// Ensure Sender implements io.Writer
var _ Sender = (Sender)(nil)

// =============================================================================
// JSONSender JSON 发送器接口
// =============================================================================

// JSONSender 支持发送 JSON 消息的发送器
type JSONSender interface {
	// SendJSON 发送 JSON 编码的消息
	SendJSON(v any) error
}

// Ensure JSONSender is a subset of Sender
var _ JSONSender = (Sender)(nil)

// =============================================================================
// Sender 实现工厂函数
// =============================================================================

// NewJSONSender 创建 JSON 发送器的辅助函数
func NewJSONSender(s Sender) JSONSender {
	return s
}

// =============================================================================
// 流事件类型
// =============================================================================

// EventType 流事件类型
type EventType string

const (
	// EventText 文本事件
	EventText EventType = "text"
	// EventToolCall 工具调用事件
	EventToolCall EventType = "tool_call"
	// EventToolResult 工具结果事件
	EventToolResult EventType = "tool_result"
	// EventThinking 思考事件
	EventThinking EventType = "thinking"
	// EventInterrupt 中断事件
	EventInterrupt EventType = "interrupt"
	// EventError 错误事件
	EventError EventType = "error"
	// EventDone 完成事件
	EventDone EventType = "done"
)

// StreamEvent 流事件
type StreamEvent struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data"`
}

// NewTextEvent 创建文本事件
func NewTextEvent(content string) *StreamEvent {
	return &StreamEvent{Type: EventText, Data: content}
}

// NewToolCallEvent 创建工具调用事件
func NewToolCallEvent(name, args string) *StreamEvent {
	return &StreamEvent{
		Type: EventToolCall,
		Data: map[string]string{
			"name": name,
			"args": args,
		},
	}
}

// NewToolResultEvent 创建工具结果事件
func NewToolResultEvent(name, result string) *StreamEvent {
	return &StreamEvent{
		Type: EventToolResult,
		Data: map[string]string{
			"name":   name,
			"result": result,
		},
	}
}

// NewThinkingEvent 创建思考事件
func NewThinkingEvent(content string) *StreamEvent {
	return &StreamEvent{Type: EventThinking, Data: content}
}

// NewInterruptEvent 创建中断事件
func NewInterruptEvent(info interface{}) *StreamEvent {
	return &StreamEvent{Type: EventInterrupt, Data: info}
}

// NewErrorEvent 创建错误事件
func NewErrorEvent(err error) *StreamEvent {
	return &StreamEvent{Type: EventError, Data: err.Error()}
}

// NewDoneEvent 创建完成事件
func NewDoneEvent() *StreamEvent {
	return &StreamEvent{Type: EventDone, Data: nil}
}

// ToJSON 将事件转换为 JSON 字节
func (e *StreamEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// SendTo 发送事件到 Sender
func (e *StreamEvent) SendTo(s Sender) error {
	data, err := e.ToJSON()
	if err != nil {
		return err
	}
	_, err = s.Write(data)
	return err
}
