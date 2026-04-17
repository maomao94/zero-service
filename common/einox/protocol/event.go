// Package protocol 定义 AI Solo 前后端通信的统一流式事件协议。
//
// 设计要点：
//
//  1. 仅流式：所有对外接口都是 SSE + JSON，每一帧 data: 后面是一个完整 JSON 对象。
//     没有非流式退化路径。
//  2. 事件自描述：每个 Event 都是自带 type 的单行 JSON，解析一次就够了，不需要
//     先拿 MessageID 再拼组件树。
//  3. 与 eino ADK 解耦：本包不依赖 adk 的任何具体事件结构；adk.AgentEvent 到
//     protocol.Event 的映射放在 adapter.go 里。
//
// 事件序列示例（单轮对话 + 一次工具调用 + 一次审批中断）：
//
//	turn.start
//	  message.start (role=assistant)
//	    message.delta ("我")
//	    message.delta ("来")
//	    message.delta ("算一下")
//	  message.end (text="我来算一下")
//	  tool.call.start (tool=calculator, args_json=...)
//	  tool.call.end   (tool=calculator, result="42")
//	  interrupt (kind=approval, ...)
//	turn.end (has_interrupt=true)
package protocol

import (
	"encoding/json"
)

// EventType 枚举所有事件类型。字符串形式便于调试/抓包。
type EventType string

const (
	EventTurnStart     EventType = "turn.start"
	EventTurnEnd       EventType = "turn.end"
	EventMessageStart  EventType = "message.start"
	EventMessageDelta  EventType = "message.delta"
	EventMessageEnd    EventType = "message.end"
	EventToolCallStart EventType = "tool.call.start"
	EventToolCallEnd   EventType = "tool.call.end"
	EventInterrupt     EventType = "interrupt"
	EventError         EventType = "error"
)

// MessageRole 与 LLM 世界保持一致的角色枚举。
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
	RoleSystem    MessageRole = "system"
)

// InterruptKind 中断交互种类，前端据此选择渲染组件。
type InterruptKind string

const (
	InterruptApproval     InterruptKind = "approval"      // 二选一审批
	InterruptSingleSelect InterruptKind = "single_select" // 单选
	InterruptMultiSelect  InterruptKind = "multi_select"  // 多选
	InterruptFreeText     InterruptKind = "free_text"     // 自由文本输入
	InterruptFormInput    InterruptKind = "form_input"    // 结构化表单
	InterruptInfoAck      InterruptKind = "info_ack"      // 展示信息 + 确认
)

// Event 是协议里每一帧的顶层结构。Data 按 Type 分别解码。
type Event struct {
	Type      EventType       `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	TurnID    string          `json:"turn_id,omitempty"`
	Seq       int64           `json:"seq"`
	Timestamp int64           `json:"ts"` // 毫秒
	Data      json.RawMessage `json:"data,omitempty"`
}

// =============================================================================
// 各事件的 Data 结构
// =============================================================================

// TurnStartData 一轮开始。
type TurnStartData struct {
	UserMessage string `json:"user_message,omitempty"`
}

// TurnEndData 一轮结束。
type TurnEndData struct {
	HasInterrupt bool   `json:"has_interrupt,omitempty"`
	InterruptID  string `json:"interrupt_id,omitempty"` // 冗余字段，便于前端直接取
	LastMessage  string `json:"last_message,omitempty"` // 本轮最后一条 assistant 文本
}

// MessageStartData 新消息开始。
type MessageStartData struct {
	MessageID string      `json:"message_id"`
	Role      MessageRole `json:"role"`
}

// MessageDeltaData 消息文本增量。
type MessageDeltaData struct {
	MessageID string `json:"message_id"`
	Text      string `json:"text"`
}

// MessageEndData 消息结束，携带完整文本。
type MessageEndData struct {
	MessageID string      `json:"message_id"`
	Role      MessageRole `json:"role"`
	Text      string      `json:"text"`
}

// ToolCallStartData 工具调用开始。
type ToolCallStartData struct {
	CallID    string `json:"call_id"`
	Tool      string `json:"tool"`
	ArgsJSON  string `json:"args_json,omitempty"`
	MessageID string `json:"message_id,omitempty"` // 关联到哪条 assistant 消息
}

// ToolCallEndData 工具调用结束。
type ToolCallEndData struct {
	CallID string `json:"call_id"`
	Tool   string `json:"tool"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// InterruptData 中断，要求用户介入。
//
// Kind 决定 Data 里哪些可选字段有效：
//   - approval     : Question / Detail
//   - single_select: Question / Options
//   - multi_select : Question / Options / MinSelect / MaxSelect
//   - free_text    : Question / Placeholder / Multiline
//   - form_input   : Question / Fields
//   - info_ack     : Title / Body（markdown）
type InterruptData struct {
	InterruptID string        `json:"interrupt_id"`
	Kind        InterruptKind `json:"kind"`
	ToolName    string        `json:"tool_name,omitempty"`
	Required    bool          `json:"required,omitempty"`

	Question    string   `json:"question,omitempty"`
	Detail      string   `json:"detail,omitempty"`
	Options     []Option `json:"options,omitempty"`
	MinSelect   int      `json:"min_select,omitempty"`
	MaxSelect   int      `json:"max_select,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Multiline   bool     `json:"multiline,omitempty"`
	Fields      []Field  `json:"fields,omitempty"`
	Title       string   `json:"title,omitempty"`
	Body        string   `json:"body,omitempty"`
}

// Option 选项（单选/多选用）。
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Desc  string `json:"desc,omitempty"`
}

// Field 表单字段（form_input 用）。
type Field struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // string | number | boolean
	Required    bool   `json:"required,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     string `json:"default,omitempty"`
}

// ErrorData 错误事件。
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
