package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpc"
)

const (
	maxResponseBodyLen = 10 << 20 // 10MB
	DefaultTimeout     = 30 * time.Second
)

type engine interface {
	Do(req *http.Request) (*http.Response, error)
}

type defaultEngine struct {
	client *http.Client
}

func (e *defaultEngine) Do(req *http.Request) (*http.Response, error) {
	return e.client.Do(req)
}

type httpcEngine struct {
	svc httpc.Service
}

func (e *httpcEngine) Do(req *http.Request) (*http.Response, error) {
	return e.svc.DoRequest(req)
}

type ClientOption func(*Client)

type Client struct {
	engine    engine
	httpcSvc  httpc.Service
	timeout   time.Duration
	tlsConfig *tls.Config
	headers   http.Header
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		timeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.engine == nil {
		c.engine = c.buildEngine()
	}
	return c
}

func (c *Client) buildEngine() engine {
	if c.httpcSvc != nil {
		return &httpcEngine{svc: c.httpcSvc}
	}

	transport := &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	if c.tlsConfig != nil {
		transport.TLSClientConfig = c.tlsConfig
	}
	return &defaultEngine{
		client: &http.Client{
			Timeout:   c.timeout,
			Transport: transport,
		},
	}
}

func WithHttpcService(svc httpc.Service) ClientOption {
	return func(c *Client) { c.httpcSvc = svc }
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) { c.timeout = d }
}

func WithTLSConfig(cfg *tls.Config) ClientOption {
	return func(c *Client) { c.tlsConfig = cfg }
}

func WithDefaultHeaders(h http.Header) ClientOption {
	return func(c *Client) { c.headers = h }
}

// --- 核心请求方法 ---

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.URL == "" {
		return nil, errors.New("request url is required")
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	bodyReader, contentType := c.buildBody(req)

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.applyHeaders(httpReq, req, contentType)
	c.applyQueryParams(httpReq, req)

	start := time.Now()
	resp, err := c.engine.Do(httpReq)
	return buildResponse(resp, err, start)
}

func (c *Client) buildBody(req *Request) (io.Reader, string) {
	if req.FormData != nil {
		return strings.NewReader(req.FormData.Encode()), "application/x-www-form-urlencoded"
	}
	if len(req.Body) == 0 {
		return nil, ""
	}

	ct := ""
	if req.Headers != nil {
		ct = strings.ToLower(req.Headers.Get("Content-Type"))
	}

	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		if encoded, err := EncodeURLEncoded(req.Body); err == nil {
			return strings.NewReader(encoded), "application/x-www-form-urlencoded"
		}
		return bytes.NewReader(req.Body), "application/x-www-form-urlencoded"
	}

	if strings.Contains(ct, "multipart/form-data") {
		data, err := ValidateAndFlatten(req.Body)
		if err == nil && len(data) > 0 {
			reader, mct, mErr := EncodeMultipart(data)
			if mErr == nil {
				return reader, mct
			}
		}
		return bytes.NewReader(req.Body), ct
	}

	return bytes.NewReader(req.Body), ""
}

func (c *Client) applyHeaders(httpReq *http.Request, req *Request, contentType string) {
	for k, vs := range c.headers {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}
	for k, vs := range req.Headers {
		for _, v := range vs {
			httpReq.Header.Set(k, v)
		}
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	} else if httpReq.Header.Get("Content-Type") == "" && len(req.Body) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}
}

func (c *Client) applyQueryParams(httpReq *http.Request, req *Request) {
	if req.QueryParams == nil {
		return
	}
	q := httpReq.URL.Query()
	for k, vs := range req.QueryParams {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	httpReq.URL.RawQuery = q.Encode()
}

// --- HTTP 便捷方法 ---

func (c *Client) Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodGet, opts...))
}

func (c *Client) Post(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPost, opts...))
}

func (c *Client) Put(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPut, opts...))
}

func (c *Client) Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodDelete, opts...))
}

func (c *Client) Patch(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPatch, opts...))
}

func (c *Client) Head(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodHead, opts...))
}

func (c *Client) Options(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodOptions, opts...))
}

// --- 文件上传 ---

func (c *Client) Upload(ctx context.Context, url string, files []FileUpload, fields map[string]string, opts ...RequestOption) (*Response, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for _, f := range files {
		part, err := w.CreateFormFile(f.FieldName, f.FileName)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err = io.Copy(part, f.Content); err != nil {
			return nil, fmt.Errorf("copy file content: %w", err)
		}
	}

	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("write field: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req := NewRequest(url, http.MethodPost, opts...)
	req.Body = buf.Bytes()
	if req.Headers == nil {
		req.Headers = make(http.Header)
	}
	req.Headers.Set("Content-Type", w.FormDataContentType())
	return c.Do(ctx, req)
}

func (c *Client) UploadFile(ctx context.Context, url, filePath, fieldName string, fields map[string]string, opts ...RequestOption) (*Response, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return c.Upload(ctx, url, []FileUpload{
		{FieldName: fieldName, FileName: filepath.Base(filePath), Content: f},
	}, fields, opts...)
}

func (c *Client) UploadBytes(ctx context.Context, url, fieldName, fileName string, data []byte, fields map[string]string, opts ...RequestOption) (*Response, error) {
	return c.Upload(ctx, url, []FileUpload{
		{FieldName: fieldName, FileName: fileName, Content: bytes.NewReader(data)},
	}, fields, opts...)
}

// --- 文件下载 ---

func (c *Client) Download(ctx context.Context, url string, opts ...RequestOption) (io.ReadCloser, error) {
	req := NewRequest(url, http.MethodGet, opts...)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.applyHeaders(httpReq, req, "")
	c.applyQueryParams(httpReq, req)

	resp, err := c.engine.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (c *Client) DownloadFile(ctx context.Context, url, destPath string, opts ...RequestOption) error {
	body, err := c.Download(ctx, url, opts...)
	if err != nil {
		return err
	}
	defer body.Close()

	dir := filepath.Dir(destPath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err = io.Copy(f, body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (c *Client) DownloadBytes(ctx context.Context, url string, opts ...RequestOption) ([]byte, error) {
	body, err := c.Download(ctx, url, opts...)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

// --- 包级便捷函数 ---

var defaultClient = NewClient()

func Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Get(ctx, url, opts...)
}

func Post(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Post(ctx, url, opts...)
}

func Put(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Put(ctx, url, opts...)
}

func Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Delete(ctx, url, opts...)
}

func Patch(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Patch(ctx, url, opts...)
}

func Head(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Head(ctx, url, opts...)
}

func SendRequest(ctx context.Context, req *Request, opts ...ClientOption) (*Response, error) {
	if len(opts) == 0 {
		return defaultClient.Do(ctx, req)
	}
	c := NewClient(opts...)
	return c.Do(ctx, req)
}

// --- 响应构建 ---

func buildResponse(resp *http.Response, err error, start time.Time) (*Response, error) {
	if err != nil {
		result := &Response{
			CostMs: time.Since(start).Milliseconds(),
			Error:  err.Error(),
		}
		result.CostFormatted = FormatCostMs(result.CostMs)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			result.StatusCode = http.StatusRequestTimeout
		} else {
			result.StatusCode = http.StatusBadGateway
		}
		return result, nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyLen+1))
	if err != nil {
		costMs := time.Since(start).Milliseconds()
		return &Response{
			StatusCode:    resp.StatusCode,
			Error:         err.Error(),
			CostMs:        costMs,
			CostFormatted: FormatCostMs(costMs),
		}, nil
	}
	if len(data) > maxResponseBodyLen {
		costMs := time.Since(start).Milliseconds()
		return &Response{
			StatusCode:    http.StatusBadGateway,
			Error:         "response body too large",
			CostMs:        costMs,
			CostFormatted: FormatCostMs(costMs),
		}, nil
	}

	costMs := time.Since(start).Milliseconds()
	return &Response{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header,
		Data:          data,
		CostMs:        costMs,
		CostFormatted: FormatCostMs(costMs),
		Success:       resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}
