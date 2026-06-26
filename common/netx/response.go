package netx

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrResponseTooLarge 表示响应体超过了配置的大小限制。
	ErrResponseTooLarge = errors.New("response body too large")
	// ErrUploadTooLarge 表示上传内容超过了配置的大小限制。
	ErrUploadTooLarge = errors.New("upload body too large")
)

// classifyNetErr 将网络层错误映射到语义对齐的 HTTP 状态码：
//   - context.DeadlineExceeded → 504 Gateway Timeout
//   - context.Canceled        → 400 Bad Request（调用方主动取消）
//   - 其他网络错误             → 503 Service Unavailable
func classifyNetErr(err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}
	if errors.Is(err, context.Canceled) {
		return http.StatusBadRequest
	}
	return http.StatusServiceUnavailable
}

// Response 封装 HTTP 响应结果，包含状态码、响应头、响应体、耗时和错误信息。
type Response struct {
	StatusCode    int
	Headers       http.Header
	Data          []byte
	CostMs        int64
	CostFormatted string
	Success       bool
	Err           error
}

// JSON 将响应体作为 JSON 反序列化到 target。
func (r *Response) JSON(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	return json.Unmarshal(r.Data, target)
}

// XML 将响应体作为 XML 反序列化到 target。
func (r *Response) XML(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	return xml.Unmarshal(r.Data, target)
}

// Text 将响应体作为纯文本字符串返回。
func (r *Response) Text() (string, error) {
	if err := r.ensureDecodable(); err != nil {
		return "", err
	}
	return string(r.Data), nil
}

// Decode 根据 Content-Type 或 body 前缀自动选择 JSON/XML/Text 进行反序列化。
func (r *Response) Decode(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	mediaType := ""
	if r.Headers != nil {
		mediaType = r.Headers.Get("Content-Type")
	}
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	mediaType = strings.ToLower(mediaType)
	sniffed := strings.TrimSpace(string(r.Data))
	if strings.Contains(mediaType, "json") || strings.HasPrefix(sniffed, "{") || strings.HasPrefix(sniffed, "[") {
		return json.Unmarshal(r.Data, target)
	}
	if strings.Contains(mediaType, "xml") || strings.HasPrefix(sniffed, "<") {
		return xml.Unmarshal(r.Data, target)
	}
	if strings.HasPrefix(mediaType, "text/") {
		if s, ok := target.(*string); ok {
			*s = string(r.Data)
			return nil
		}
		return fmt.Errorf("decode text response requires *string target")
	}
	return fmt.Errorf("unsupported response content type: %s", mediaType)
}

func (r *Response) ensureDecodable() error {
	if r == nil {
		return errors.New("response is nil")
	}
	if r.Err != nil {
		return r.Err
	}
	if !r.Success {
		return fmt.Errorf("request failed: status %d", r.StatusCode)
	}
	return nil
}

// FormatCostMs 将毫秒耗时格式化为可读字符串（如 "150ms" 或 "1.5s"）。
func FormatCostMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func elapsedSince(start time.Time) (int64, string) {
	costMs := time.Since(start).Milliseconds()
	return costMs, FormatCostMs(costMs)
}
