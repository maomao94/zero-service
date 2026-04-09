package ssex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Writer 封装 SSE 协议写入，自动 Flush
// 支持 io.Writer 接口，用于流式 A2UI 输出
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	buf     []byte // 缓冲，用于 io.Writer 实现
}

// NewWriter 创建 SSE Writer，要求 ResponseWriter 支持 Flusher
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("ssex: streaming not supported")
	}
	return &Writer{w: w, flusher: flusher}, nil
}

// Write 实现 io.Writer 接口
// 将数据缓冲，直到遇到换行符，每行作为 SSE data 事件发送
func (w *Writer) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := w.buf[:idx]
		w.buf = w.buf[idx+1:]
		if len(line) == 0 {
			continue
		}
		fmt.Fprintf(w.w, "data: %s\n\n", line)
		w.flusher.Flush()
	}
	return len(p), nil
}

// WriteEvent 写入带事件名的消息
//
//	event: {event}\n
//	data: {data}\n
//	\n
func (w *Writer) WriteEvent(event, data string) {
	fmt.Fprintf(w.w, "event: %s\ndata: %s\n\n", event, data)
	w.flusher.Flush()
}

// WriteData 写入纯数据消息
//
//	data: {data}\n
//	\n
func (w *Writer) WriteData(data string) {
	fmt.Fprintf(w.w, "data: %s\n\n", data)
	w.flusher.Flush()
}

// WriteComment 写入注释行（客户端会忽略）
//
//	: {comment}\n
//	\n
func (w *Writer) WriteComment(comment string) {
	fmt.Fprintf(w.w, ": %s\n\n", comment)
	w.flusher.Flush()
}

// WriteKeepAlive 写入心跳保活注释
func (w *Writer) WriteKeepAlive() {
	w.WriteComment("keepalive")
}

// WriteJSON 将结构体序列化为 JSON 并以 data: {json} 格式写出（OpenAI SSE 标准格式）
//
//	data: {"id":"chatcmpl-xxx","choices":[...]}\n
//	\n
func (w *Writer) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Fprintf(w.w, "data: %s\n\n", data)
	w.flusher.Flush()
	return nil
}

// WriteDone 写入 OpenAI 流结束标记
//
//	data: [DONE]\n
//	\n
func (w *Writer) WriteDone() {
	fmt.Fprint(w.w, "data: [DONE]\n\n")
	w.flusher.Flush()
}

// Flush 手动刷新
func (w *Writer) Flush() {
	w.flusher.Flush()
}

// BufferFlush 刷新缓冲区中剩余的数据
func (w *Writer) BufferFlush() {
	if len(w.buf) > 0 {
		fmt.Fprintf(w.w, "data: %s\n\n", w.buf)
		w.buf = nil
		w.flusher.Flush()
	}
}

// ResponseWriter 返回原始的 http.ResponseWriter
func (w *Writer) ResponseWriter() http.ResponseWriter {
	return w.w
}

// Ensure Writer implements io.Writer
var _ io.Writer = (*Writer)(nil)

// Backward compatibility aliases
type LineWriter = Writer

func NewLineWriter(w http.ResponseWriter) (*Writer, error) {
	return NewWriter(w)
}
