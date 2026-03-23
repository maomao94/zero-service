package provider

import "context"

// Provider 大模型 API 提供者接口
type Provider interface {
	// ChatCompletion 非流式对话补全
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatCompletionStream 流式对话补全
	ChatCompletionStream(ctx context.Context, req *ChatRequest) (StreamReader, error)
}

// StreamReader 流式响应读取器
type StreamReader interface {
	// Recv 读取下一个 chunk，io.EOF 表示结束
	Recv() (*StreamChunk, error)
	// Close 释放资源（关闭 HTTP response body 等）
	Close() error
}
