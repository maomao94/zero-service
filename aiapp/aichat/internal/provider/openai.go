package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// OpenAICompatible OpenAI 兼容协议的 provider 实现
type OpenAICompatible struct {
	endpoint string
	apiKey   string
}

// NewOpenAICompatible 创建 OpenAI 兼容 provider
func NewOpenAICompatible(endpoint, apiKey string) *OpenAICompatible {
	return &OpenAICompatible{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
	}
}

// ChatCompletion 非流式对话补全
func (o *OpenAICompatible) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	httpReq, err := o.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := httpc.DoRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, o.parseError(resp)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &chatResp, nil
}

// ChatCompletionStream 流式对话补全
func (o *OpenAICompatible) ChatCompletionStream(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	req.Stream = true

	httpReq, err := o.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := httpc.DoRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, o.parseError(resp)
	}

	return &sseStreamReader{
		scanner: bufio.NewScanner(resp.Body),
		body:    resp.Body,
	}, nil
}

// buildRequest 构建 HTTP 请求
func (o *OpenAICompatible) buildRequest(ctx context.Context, req *ChatRequest) (*http.Request, error) {
	body, err := marshalWithExtraBody(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	return httpReq, nil
}

// marshalWithExtraBody 序列化请求体，将 ExtraBody 中的厂商扩展参数合并到 JSON 顶层。
//
// 工作流程：
//  1. 先用标准 json.Marshal 序列化 ChatRequest（ExtraBody 因 json:"-" 被忽略）
//  2. 若 ExtraBody 非空，反序列化为 map，将 ExtraBody 的 key-value 逐一合并到顶层
//  3. 重新序列化为最终 JSON 发送给厂商 API
//
// 这样既保留了标准 OpenAI 字段，又能注入厂商特有参数（如千问的 enable_thinking、智谱的 thinking 对象），
// 实现了一套代码兼容多个大模型厂商的扩展需求。
func marshalWithExtraBody(req *ChatRequest) ([]byte, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if len(req.ExtraBody) == 0 {
		return data, nil
	}
	// 反序列化为 map，合并 ExtraBody，再序列化
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for k, v := range req.ExtraBody {
		m[k] = v
	}
	return json.Marshal(m)
}

// parseError 解析大模型 API 的错误响应
func (o *OpenAICompatible) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// APIError 大模型 API 错误
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (status %d): %s", e.StatusCode, e.Body)
}

// sseStreamReader 基于 SSE 的流式读取器
type sseStreamReader struct {
	scanner *bufio.Scanner
	body    io.ReadCloser
}

func (r *sseStreamReader) Recv() (*StreamChunk, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 data: 前缀
		if !strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)

		// [DONE] 标记流结束
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("unmarshal chunk: %w", err)
		}

		return &chunk, nil
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	// scanner 结束但没收到 [DONE]，视为正常结束
	return nil, io.EOF
}

func (r *sseStreamReader) Close() error {
	return r.body.Close()
}
