package builtin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// =============================================================================
// now —— 返回服务器当前时间
// =============================================================================

type nowParam struct {
	Format string `json:"format,omitempty" jsonschema:"description=时间格式，遵循 Go time.Format 规则; 为空则返回 RFC3339"`
}

type nowResult struct {
	Time string `json:"time"`
	Unix int64  `json:"unix"`
}

// NewNowTool 返回一个获取当前时间的工具。
func NewNowTool() (tool.InvokableTool, error) {
	return utils.InferTool("now", "Now: 返回服务器当前时间 (RFC3339 或指定 Go 格式)。",
		func(_ context.Context, in *nowParam) (*nowResult, error) {
			t := time.Now()
			layout := in.Format
			if layout == "" {
				layout = time.RFC3339
			}
			return &nowResult{Time: t.Format(layout), Unix: t.Unix()}, nil
		})
}

// NewNow 返回一个获取当前时间的工具。
func NewNow() tool.InvokableTool {
	t, err := NewNowTool()
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// random_id —— 返回一段随机 hex ID
// =============================================================================

type randIDParam struct {
	Bytes int `json:"bytes,omitempty" jsonschema:"description=随机字节数, 默认 8 (=16 位 hex)"`
}

type randIDResult struct {
	ID string `json:"id"`
}

// NewRandomIDTool 返回一个产生随机 ID 的工具。
func NewRandomIDTool() (tool.InvokableTool, error) {
	return utils.InferTool("random_id", "RandomID: 生成随机 hex ID, 用于临时命名。",
		func(_ context.Context, in *randIDParam) (*randIDResult, error) {
			n := in.Bytes
			if n <= 0 {
				n = 8
			}
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				return nil, err
			}
			return &randIDResult{ID: hex.EncodeToString(buf)}, nil
		})
}

// NewRandomID 返回一个产生随机 ID 的工具。
func NewRandomID() tool.InvokableTool {
	t, err := NewRandomIDTool()
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// http_get —— 发送 HTTP GET 请求
// =============================================================================

const maxHTTPGetBody = 512 * 1024 // 最大响应体 512KB
const httpGetTimeout = 30 * time.Second

type httpGetParam struct {
	URL     string            `json:"url" jsonschema:"required,description=请求 URL，必须包含协议前缀（http:// 或 https://）"`
	Headers map[string]string `json:"headers,omitempty" jsonschema:"description=自定义请求头，可选"`
}

type httpGetResult struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// NewHTTPGetTool 返回一个 HTTP GET 请求工具。
func NewHTTPGetTool() (tool.InvokableTool, error) {
	return utils.InferTool("http_get", "HTTP GET: 向指定 URL 发送 GET 请求并返回响应体。适用于获取网页、调用 REST API、检查端点状态。最大响应体 512KB，超时 30 秒。",
		func(_ context.Context, in *httpGetParam) (*httpGetResult, error) {
			url := strings.TrimSpace(in.URL)
			if url == "" {
				return &httpGetResult{Error: "url 为空"}, nil
			}
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				return &httpGetResult{Error: "url 必须包含协议前缀（http:// 或 https://）"}, nil
			}

			client := &http.Client{Timeout: httpGetTimeout}
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return &httpGetResult{Error: fmt.Sprintf("创建请求失败: %v", err)}, nil
			}
			for k, v := range in.Headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				return &httpGetResult{Error: fmt.Sprintf("请求失败: %v", err)}, nil
			}
			defer resp.Body.Close()

			limited := io.LimitReader(resp.Body, maxHTTPGetBody)
			bodyBytes, err := io.ReadAll(limited)
			if err != nil {
				return &httpGetResult{StatusCode: resp.StatusCode, Error: fmt.Sprintf("读取响应失败: %v", err)}, nil
			}

			resultHeaders := make(map[string]string, 4)
			for k := range resp.Header {
				if len(resultHeaders) < 8 {
					resultHeaders[k] = resp.Header.Get(k)
				}
			}

			return &httpGetResult{
				StatusCode: resp.StatusCode,
				Body:       string(bodyBytes),
				Headers:    resultHeaders,
			}, nil
		})
}

// NewHTTPGet 返回一个 HTTP GET 请求工具。
func NewHTTPGet() tool.InvokableTool {
	t, err := NewHTTPGetTool()
	if err != nil {
		panic(err)
	}
	return t
}
