package a2ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// =============================================================================
// SSE Writer
// =============================================================================

// SSEWriter SSE 格式写入器
type SSEWriter struct {
	mu     sync.RWMutex
	writer io.Writer
}

func NewSSEWriter(w io.Writer) *SSEWriter {
	return &SSEWriter{
		writer: w,
	}
}

// Write 写入 SSE 事件
func (w *SSEWriter) Write(event *Event) error {
	if w.writer == nil {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// 自动生成 ID
	if event.ID == "" {
		event.ID = uuid.NewString()
	}

	// SSE 格式：data: <json>\n\n
	_, err = w.writer.Write([]byte("data: " + string(data) + "\n\n"))
	if err != nil {
		return fmt.Errorf("write sse: %w", err)
	}

	return nil
}

// WriteWithRetry 写入并支持重试
func (w *SSEWriter) WriteWithRetry(event *Event, maxRetries int) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if err := w.Write(event); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

// Flush 刷新缓冲区
func (w *SSEWriter) Flush() error {
	if f, ok := w.writer.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// SetWriter 设置写入器
func (w *SSEWriter) SetWriter(writer io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer = writer
}

// =============================================================================
// HTTP 辅助函数
// =============================================================================

const (
	// MIMEType SSE MIME 类型
	MIMEType = "text/event-stream"
)

// SetupSSEHeaders 设置 SSE 响应头
func SetupSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", MIMEType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// IsSSEAcceptable 检查是否接受 SSE
func IsSSEAcceptable(accept string) bool {
	return accept == MIMEType || accept == "*/*" || accept == ""
}
