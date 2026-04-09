package stream

import (
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/grpc/status"
)

// =============================================================================
// GRPCSender - gRPC 流发送器
// =============================================================================

// ChunkSender gRPC 流 Chunk 发送接口
type ChunkSender interface {
	Send(chunk interface{}) error
}

// GRPCSender gRPC 流发送器实现
// 用于将流式数据发送到 gRPC 客户端
type GRPCSender struct {
	sender    ChunkSender
	sendFunc  func(interface{}) error
	sessionID string
}

// NewGRPCSender 创建 gRPC 发送器
// sendFunc: gRPC Stream.Send 方法
func NewGRPCSender(sendFunc func(interface{}) error, sessionID string) *GRPCSender {
	return &GRPCSender{
		sendFunc:  sendFunc,
		sessionID: sessionID,
	}
}

// Write 实现 io.Writer 接口
// 将数据封装为 StreamChunk 消息发送
func (s *GRPCSender) Write(p []byte) (n int, err error) {
	chunk := map[string]interface{}{
		"session_id": s.sessionID,
		"data":       string(p),
		"is_final":   false,
	}
	if err := s.sendFunc(chunk); err != nil {
		return 0, err
	}
	return len(p), nil
}

// SendJSON 发送 JSON 编码的消息
func (s *GRPCSender) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	chunk := map[string]interface{}{
		"session_id": s.sessionID,
		"data":       string(data),
		"is_final":   false,
	}
	return s.sendFunc(chunk)
}

// SendDone 发送流结束信号
func (s *GRPCSender) SendDone() {
	chunk := map[string]interface{}{
		"session_id": s.sessionID,
		"is_final":   true,
	}
	_ = s.sendFunc(chunk) // 忽略错误
}

// SendError 发送错误信号
func (s *GRPCSender) SendError(err error) {
	chunk := map[string]interface{}{
		"session_id": s.sessionID,
		"error":      err.Error(),
		"is_final":   true,
	}
	_ = s.sendFunc(chunk) // 忽略错误
}

// SessionID 返回会话 ID
func (s *GRPCSender) SessionID() string {
	return s.sessionID
}

// Ensure GRPCSender implements Sender
var _ Sender = (*GRPCSender)(nil)

// =============================================================================
// GRPCStreamChunk - gRPC 流消息结构
// =============================================================================

// GRPCStreamChunk gRPC 流消息，用于 AskStream 接口
type GRPCStreamChunk struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"`
	IsFinal   bool   `json:"is_final"`
	Error     string `json:"error,omitempty"`
}

// NewGRPCStreamChunk 创建流消息
func NewGRPCStreamChunk(sessionID, data string, isFinal bool) *GRPCStreamChunk {
	return &GRPCStreamChunk{
		SessionID: sessionID,
		Data:      data,
		IsFinal:   isFinal,
	}
}

// NewGRPCErrorChunk 创建错误消息
func NewGRPCErrorChunk(sessionID string, err error) *GRPCStreamChunk {
	errMsg := ""
	if err != nil {
		errMsg = status.Convert(err).Message()
	}
	return &GRPCStreamChunk{
		SessionID: sessionID,
		IsFinal:   true,
		Error:     errMsg,
	}
}

// ToJSON 将消息转换为 JSON
func (c *GRPCStreamChunk) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// Send 发送消息到 gRPC 流
func (c *GRPCStreamChunk) Send(sendFunc func(interface{}) error) error {
	return sendFunc(c)
}

// =============================================================================
// GRPCStreamSender - 改进版 gRPC 发送器
// =============================================================================

// GRPCStreamSender 使用 GRPCStreamChunk 的 gRPC 发送器
type GRPCStreamSender struct {
	sendFunc  func(*GRPCStreamChunk) error
	sessionID string
}

// NewGRPCStreamSender 创建 gRPC 流发送器
func NewGRPCStreamSender(sendFunc func(*GRPCStreamChunk) error, sessionID string) *GRPCStreamSender {
	return &GRPCStreamSender{
		sendFunc:  sendFunc,
		sessionID: sessionID,
	}
}

// Write 实现 io.Writer 接口
func (s *GRPCStreamSender) Write(p []byte) (n int, err error) {
	chunk := NewGRPCStreamChunk(s.sessionID, string(p), false)
	if err := s.sendFunc(chunk); err != nil {
		return 0, err
	}
	return len(p), nil
}

// SendJSON 发送 JSON 编码的消息
func (s *GRPCStreamSender) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	chunk := NewGRPCStreamChunk(s.sessionID, string(data), false)
	return s.sendFunc(chunk)
}

// SendDone 发送流结束信号
func (s *GRPCStreamSender) SendDone() {
	chunk := NewGRPCStreamChunk(s.sessionID, "", true)
	_ = s.sendFunc(chunk)
}

// SendError 发送错误信号
func (s *GRPCStreamSender) SendError(err error) {
	chunk := NewGRPCErrorChunk(s.sessionID, err)
	_ = s.sendFunc(chunk)
}

// SessionID 返回会话 ID
func (s *GRPCStreamSender) SessionID() string {
	return s.sessionID
}

// Ensure GRPCStreamSender implements Sender
var _ Sender = (*GRPCStreamSender)(nil)

// Ensure io.Writer compatibility
var _ io.Writer = (*GRPCStreamSender)(nil)

// =============================================================================
// 辅助函数
// =============================================================================

// SendStreamEvent 发送流事件
func SendStreamEvent(s Sender, event *StreamEvent) error {
	return event.SendTo(s)
}

// SendTextChunk 发送文本块
func SendTextChunk(s Sender, content string) error {
	event := NewTextEvent(content)
	return event.SendTo(s)
}

// SendErrorChunk 发送错误块
func SendErrorChunk(s Sender, err error) error {
	event := NewErrorEvent(err)
	return event.SendTo(s)
}

// SendDoneSignal 发送完成信号
func SendDoneSignal(s Sender) {
	s.SendDone()
}

// SendInterruptInfo 发送中断信息
func SendInterruptInfo(s Sender, info interface{}) error {
	event := NewInterruptEvent(info)
	return event.SendTo(s)
}

// FormatToolCall 格式化工具调用消息
func FormatToolCall(name, args string) string {
	return fmt.Sprintf("[Calling tool: %s with args: %s]", name, args)
}

// FormatToolResult 格式化工具结果消息
func FormatToolResult(name, result string) string {
	return fmt.Sprintf("[Tool %s returned: %s]", name, result)
}
