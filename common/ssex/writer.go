package ssex

import (
	"fmt"
	"net/http"
)

// Writer 封装 SSE 协议写入，自动 Flush
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewWriter 创建 SSE Writer，要求 ResponseWriter 支持 Flusher
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("ssex: streaming not supported")
	}
	return &Writer{w: w, flusher: flusher}, nil
}

// WriteEvent 写入带事件名的消息
//
//	event: {event}\n
//	data: {data}\n
//	\n
func (sw *Writer) WriteEvent(event, data string) {
	fmt.Fprintf(sw.w, "event: %s\ndata: %s\n\n", event, data)
	sw.flusher.Flush()
}

// WriteData 写入纯数据消息
//
//	data: {data}\n
//	\n
func (sw *Writer) WriteData(data string) {
	fmt.Fprintf(sw.w, "data: %s\n\n", data)
	sw.flusher.Flush()
}

// WriteComment 写入注释行（客户端会忽略）
//
//	: {comment}\n
//	\n
func (sw *Writer) WriteComment(comment string) {
	fmt.Fprintf(sw.w, ": %s\n\n", comment)
	sw.flusher.Flush()
}

// WriteKeepAlive 写入心跳保活注释
func (sw *Writer) WriteKeepAlive() {
	sw.WriteComment("keepalive")
}
