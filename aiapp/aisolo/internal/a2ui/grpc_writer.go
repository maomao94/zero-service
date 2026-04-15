package a2ui

import (
	"errors"
	"io"

	"zero-service/aiapp/aisolo/aisolo"
)

// GRPCStreamWriter 实现 io.Writer 接口，将 A2UI JSON 写入 gRPC Stream
type GRPCStreamWriter struct {
	stream    aisolo.AiSolo_AskStreamServer
	sessionID string
}

// NewGRPCStreamWriter 创建 GRPCStreamWriter
func NewGRPCStreamWriter(stream aisolo.AiSolo_AskStreamServer, sessionID string) *GRPCStreamWriter {
	return &GRPCStreamWriter{
		stream:    stream,
		sessionID: sessionID,
	}
}

// Write 实现 io.Writer 接口
func (w *GRPCStreamWriter) Write(p []byte) (n int, err error) {
	if w.stream == nil {
		return 0, errors.New("stream is nil")
	}
	resp := &aisolo.AskStreamResp{
		Chunk: &aisolo.AskStreamChunk{
			SessionId: w.sessionID,
			Data:      string(p),
			IsFinal:   false,
		},
	}
	if err := w.stream.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

// SessionID 返回会话 ID
func (w *GRPCStreamWriter) SessionID() string {
	return w.sessionID
}

// Ensure GRPCStreamWriter implements io.Writer
var _ io.Writer = (*GRPCStreamWriter)(nil)
