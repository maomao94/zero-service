package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

// =============================================================================
// SSE + JSON 编码
// =============================================================================
//
// 每个事件按 SSE 规范输出：
//
//	data: <single-line JSON>\n
//	\n
//
// 前端用 EventSource 直接读取即可；每个 SSE `data:` 段的 payload 是完整 JSON
// 对象（不是多行拼接），用 JSON.parse 一次就能得到 Event。

// Encode 把 Event 编码成一行 JSON（不含 SSE 包装），末尾带 \n。
// 用于单测、gRPC 透传等场景。
func Encode(e Event) ([]byte, error) {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().UnixMilli()
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(e); err != nil {
		return nil, fmt.Errorf("protocol: encode: %w", err)
	}
	return buf.Bytes(), nil
}

// EncodeSSE 把 Event 编码成 SSE 帧（`data: <json>\n\n`）。
func EncodeSSE(e Event) ([]byte, error) {
	line, err := Encode(e)
	if err != nil {
		return nil, err
	}
	// Encode 返回末尾带 \n，这里去掉再包 SSE 规范格式
	line = bytes.TrimRight(line, "\n")
	buf := &bytes.Buffer{}
	buf.WriteString("data: ")
	buf.Write(line)
	buf.WriteString("\n\n")
	return buf.Bytes(), nil
}

// Decode 解码一行 JSON（不处理 SSE 包装）。
func Decode(line []byte) (Event, error) {
	var e Event
	if err := json.Unmarshal(bytes.TrimSpace(line), &e); err != nil {
		return e, fmt.Errorf("protocol: decode: %w", err)
	}
	return e, nil
}

// DecodeSSE 从 SSE 帧（可能多行）中取出 data 字段并解码为 Event。
// 支持多条 data: 行，会按 SSE 规范用换行拼接。
func DecodeSSE(frame []byte) (Event, error) {
	var sb strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(frame))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(line[len("data:"):])
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(payload)
	}
	if err := scanner.Err(); err != nil {
		return Event{}, fmt.Errorf("protocol: decode sse: %w", err)
	}
	if sb.Len() == 0 {
		return Event{}, errors.New("protocol: empty sse frame")
	}
	return Decode([]byte(sb.String()))
}

// =============================================================================
// Emitter：按轮次递增 seq 的写入器
// =============================================================================

// Emitter 是一个轮次级的事件写入器。内部维护 seq、sessionID、turnID，
// 业务层只关心 emit 什么事件，不需要每次都填元数据。
//
// 线程不安全：一个 Emitter 对应一次 Run，单 goroutine 使用。
type Emitter struct {
	w         io.Writer
	sessionID string
	turnID    string
	seq       int64
	sse       bool // true => SSE 帧；false => 裸 JSON 行
}

// NewEmitter 创建裸 JSON 行（NDJSON）写入器。
func NewEmitter(w io.Writer, sessionID, turnID string) *Emitter {
	return &Emitter{w: w, sessionID: sessionID, turnID: turnID}
}

// NewSSEEmitter 创建 SSE 帧写入器（用于 HTTP 网关）。
func NewSSEEmitter(w io.Writer, sessionID, turnID string) *Emitter {
	return &Emitter{w: w, sessionID: sessionID, turnID: turnID, sse: true}
}

// Session / Turn 只读访问。
func (e *Emitter) Session() string { return e.sessionID }
func (e *Emitter) Turn() string    { return e.turnID }

// Emit 写一个事件。Data 会被 JSON 编码进 Event.Data。
func (e *Emitter) Emit(t EventType, data any) error {
	raw, err := marshalData(data)
	if err != nil {
		return err
	}
	ev := Event{
		Type:      t,
		SessionID: e.sessionID,
		TurnID:    e.turnID,
		Seq:       atomic.AddInt64(&e.seq, 1) - 1,
		Timestamp: time.Now().UnixMilli(),
		Data:      raw,
	}

	var buf []byte
	if e.sse {
		buf, err = EncodeSSE(ev)
	} else {
		buf, err = Encode(ev)
	}
	if err != nil {
		return err
	}
	if _, err := e.w.Write(buf); err != nil {
		return fmt.Errorf("protocol: write: %w", err)
	}
	return nil
}

// EmitError 便捷方法。
func (e *Emitter) EmitError(code, msg string) error {
	return e.Emit(EventError, ErrorData{Code: code, Message: msg})
}

// TurnStart 一轮开始（可携带技能命中信息）。
func (e *Emitter) TurnStart(data TurnStartData) error {
	return e.Emit(EventTurnStart, data)
}

func (e *Emitter) TurnEnd(hasInterrupt bool, interruptID, lastMsg string) error {
	return e.Emit(EventTurnEnd, TurnEndData{
		HasInterrupt: hasInterrupt,
		InterruptID:  interruptID,
		LastMessage:  lastMsg,
	})
}

func marshalData(data any) (json.RawMessage, error) {
	if data == nil {
		return nil, nil
	}
	if raw, ok := data.(json.RawMessage); ok {
		return raw, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("protocol: marshal data: %w", err)
	}
	return b, nil
}
