package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultMaxResponseBytes 默认响应体最大字节数（10MB）。
	DefaultMaxResponseBytes = 10 << 20
	// DefaultUploadBytesLimit 默认上传内容最大字节数（32MB）。
	DefaultUploadBytesLimit = 32 << 20
)

// ClientOptions 表示 Client 构造配置。
type ClientOptions struct {
	Engine             Engine
	TLSConfig          *tls.Config
	Headers            http.Header
	MaxResponseBytes   int64
	DownloadBytesLimit int64
	UploadBytesLimit   int64
	HTTPClientOptions  []func(*http.Client)
}

// ClientOption 函数式客户端配置选项。
type ClientOption func(*ClientOptions)

// Client 是 HTTP 客户端，支持引擎抽象、TLS 配置、请求/响应/下载/上传大小限制等。
type Client struct {
	engine             Engine
	tlsConfig          *tls.Config
	headers            http.Header
	maxResponseBytes   int64
	downloadBytesLimit int64
	uploadBytesLimit   int64
}

// NewClient 创建 Client，若不指定引擎则使用 DefaultEngine。
func NewClient(opts ...ClientOption) *Client {
	o := &ClientOptions{
		MaxResponseBytes:   DefaultMaxResponseBytes,
		DownloadBytesLimit: DefaultDownloadBytesLimit,
		UploadBytesLimit:   DefaultUploadBytesLimit,
	}
	for _, opt := range opts {
		opt(o)
	}
	c := &Client{
		engine:             o.Engine,
		tlsConfig:          o.TLSConfig,
		headers:            o.Headers,
		maxResponseBytes:   o.MaxResponseBytes,
		downloadBytesLimit: o.DownloadBytesLimit,
		uploadBytesLimit:   o.UploadBytesLimit,
	}
	if c.engine == nil {
		c.engine = c.buildEngine(o.HTTPClientOptions)
	}
	return c
}

func (c *Client) buildEngine(httpClientOpts []func(*http.Client)) Engine {
	httpClient := newHTTPClient(WithTransportTLS(c.tlsConfig))
	for _, fn := range httpClientOpts {
		fn(httpClient)
	}
	return &DefaultEngine{client: httpClient}
}

// WithEngine 设置底层 HTTP 执行引擎。
func WithEngine(e Engine) ClientOption {
	return func(o *ClientOptions) { o.Engine = e }
}

// WithTLSConfig 设置 TLS 配置。
func WithTLSConfig(cfg *tls.Config) ClientOption {
	return func(o *ClientOptions) { o.TLSConfig = cfg }
}

// WithDefaultHeaders 设置全局默认请求头（会 Clone，避免外部修改影响）。
func WithDefaultHeaders(h http.Header) ClientOption {
	return func(o *ClientOptions) { o.Headers = h.Clone() }
}

// WithMaxResponseBytes 设置响应体最大字节数限制，设为 0 则不限制。
func WithMaxResponseBytes(maxBytes int64) ClientOption {
	return func(o *ClientOptions) { o.MaxResponseBytes = maxBytes }
}

// WithDownloadBytesLimit 设置下载最大字节数限制，设为 0 则不限制。
func WithDownloadBytesLimit(maxBytes int64) ClientOption {
	return func(o *ClientOptions) { o.DownloadBytesLimit = maxBytes }
}

// WithUploadBytesLimit 设置上传最大字节数限制，设为 0 则不限制。
func WithUploadBytesLimit(maxBytes int64) ClientOption {
	return func(o *ClientOptions) { o.UploadBytesLimit = maxBytes }
}

// WithHTTPClientOption 透传配置到底层 http.Client（仅在未自定义 Engine 时生效）。
// 可用于设置 CheckRedirect、Jar、Timeout 等标准库 http.Client 字段。
func WithHTTPClientOption(fn func(*http.Client)) ClientOption {
	return func(o *ClientOptions) { o.HTTPClientOptions = append(o.HTTPClientOptions, fn) }
}

// Do 执行 HTTP 请求，返回统一的 Response 结构。
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.URL == "" {
		return nil, errors.New("request url is required")
	}
	if req.OptionError != nil {
		return nil, req.OptionError
	}
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	bodyReader, contentType := c.buildBody(req)
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.applyHeaders(httpReq, req, contentType)
	c.applyQueryParams(httpReq, req)
	start := time.Now()
	resp, err := c.engine.Do(httpReq)
	return c.buildResponse(resp, err, start)
}

func (c *Client) buildBody(req *Request) (io.Reader, string) {
	if req.FormData != nil || req.bodyKind == bodyKindForm {
		values := req.FormData
		if values == nil {
			values = make(url.Values)
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded"
	}
	if req.BodyReader != nil || req.bodyKind == bodyKindReader {
		return req.BodyReader, req.ContentType
	}
	if req.bodyKind == bodyKindJSON {
		return bytes.NewReader(req.Body), req.ContentType
	}
	if len(req.Body) == 0 {
		return nil, ""
	}
	return c.buildRawBody(req)
}

func (c *Client) buildRawBody(req *Request) (io.Reader, string) {
	ct := req.ContentType
	if req.Headers != nil && req.Headers.Get("Content-Type") != "" {
		ct = req.Headers.Get("Content-Type")
	}
	if strings.Contains(strings.ToLower(ct), "application/x-www-form-urlencoded") {
		return EncodeURLEncodedIfNeeded(req.Body)
	}
	return bytes.NewReader(req.Body), ct
}

func (c *Client) applyHeaders(httpReq *http.Request, req *Request, contentType string) {
	for k, vs := range c.headers {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}
	for k, vs := range req.Headers {
		httpReq.Header.Del(k)
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}
	if contentType != "" {
		if strings.Contains(strings.ToLower(contentType), "multipart/form-data") {
			httpReq.Header.Set("Content-Type", contentType)
			return
		}
		if httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", contentType)
		}
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

// Get 发送 GET 请求。
func (c *Client) Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodGet, opts...))
}

// Post 发送 POST 请求。
func (c *Client) Post(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPost, opts...))
}

// Put 发送 PUT 请求。
func (c *Client) Put(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPut, opts...))
}

// Delete 发送 DELETE 请求。
func (c *Client) Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodDelete, opts...))
}

// Patch 发送 PATCH 请求。
func (c *Client) Patch(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodPatch, opts...))
}

// Head 发送 HEAD 请求。
func (c *Client) Head(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodHead, opts...))
}

// Options 发送 OPTIONS 请求。
func (c *Client) Options(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, NewRequest(url, http.MethodOptions, opts...))
}

func (c *Client) buildResponse(resp *http.Response, err error, start time.Time) (*Response, error) {
	costMs, costFormatted := elapsedSince(start)
	if err != nil {
		return &Response{
			StatusCode:    classifyNetErr(err),
			CostMs:        costMs,
			CostFormatted: costFormatted,
			Err:           err,
		}, nil
	}
	defer resp.Body.Close()
	var data []byte
	var readErr error
	if c.maxResponseBytes > 0 {
		data, readErr = io.ReadAll(http.MaxBytesReader(nil, resp.Body, c.maxResponseBytes))
	} else {
		data, readErr = io.ReadAll(resp.Body)
	}
	if readErr != nil {
		return &Response{
			StatusCode:    http.StatusBadGateway,
			Err:           fmt.Errorf("%w: limit %d bytes", ErrResponseTooLarge, c.maxResponseBytes),
			CostMs:        costMs,
			CostFormatted: costFormatted,
		}, nil
	}
	return &Response{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header.Clone(),
		Data:          data,
		CostMs:        costMs,
		CostFormatted: costFormatted,
		Success:       resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}
