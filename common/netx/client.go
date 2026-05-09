package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultMaxResponseBytes = 10 << 20
	DefaultUploadBytesLimit = 32 << 20
)

type ClientOption func(*Client)

type Client struct {
	engine             Engine
	tlsConfig          *tls.Config
	headers            http.Header
	maxResponseBytes   int64
	downloadBytesLimit int64
	uploadBytesLimit   int64
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		maxResponseBytes:   DefaultMaxResponseBytes,
		downloadBytesLimit: DefaultDownloadBytesLimit,
		uploadBytesLimit:   DefaultUploadBytesLimit,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.engine == nil {
		c.engine = c.buildEngine()
	}
	return c
}

func (c *Client) buildEngine() Engine {
	return &DefaultEngine{client: newHTTPClient(WithTransportTLS(c.tlsConfig))}
}

func WithEngine(e Engine) ClientOption {
	return func(c *Client) { c.engine = e }
}

func WithTLSConfig(cfg *tls.Config) ClientOption {
	return func(c *Client) { c.tlsConfig = cfg }
}

func WithDefaultHeaders(h http.Header) ClientOption {
	return func(c *Client) { c.headers = h.Clone() }
}

func WithMaxResponseBytes(maxBytes int64) ClientOption {
	return func(c *Client) { c.maxResponseBytes = maxBytes }
}

func WithDownloadBytesLimit(maxBytes int64) ClientOption {
	return func(c *Client) { c.downloadBytesLimit = maxBytes }
}

func WithUploadBytesLimit(maxBytes int64) ClientOption {
	return func(c *Client) { c.uploadBytesLimit = maxBytes }
}

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
	switch req.bodyKind {
	case bodyKindForm:
		return strings.NewReader(req.FormData.Encode()), req.ContentType
	case bodyKindJSON:
		return bytes.NewReader(req.Body), req.ContentType
	case bodyKindReader:
		return req.BodyReader, req.ContentType
	case bodyKindRaw:
		return c.buildRawBody(req)
	}
	if req.FormData != nil {
		return strings.NewReader(req.FormData.Encode()), "application/x-www-form-urlencoded"
	}
	if req.BodyReader != nil {
		return req.BodyReader, req.ContentType
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

func (c *Client) buildResponse(resp *http.Response, err error, start time.Time) (*Response, error) {
	costMs, costFormatted := elapsedSince(start)
	if err != nil {
		result := &Response{CostMs: costMs, CostFormatted: costFormatted, Err: err}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context deadline exceeded") {
			result.StatusCode = http.StatusRequestTimeout
		} else {
			result.StatusCode = http.StatusBadGateway
		}
		return result, nil
	}
	defer resp.Body.Close()
	if c.maxResponseBytes <= 0 {
		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return &Response{StatusCode: resp.StatusCode, Err: readErr, CostMs: costMs, CostFormatted: costFormatted}, nil
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
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if readErr != nil {
		return &Response{StatusCode: resp.StatusCode, Err: readErr, CostMs: costMs, CostFormatted: costFormatted}, nil
	}
	if int64(len(data)) > c.maxResponseBytes {
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
