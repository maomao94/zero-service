package netx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const DefaultDownloadBytesLimit = 32 << 20

type DownloadOption func(*downloadOptions)

type downloadOptions struct {
	maxBytes int64
	rangeSet bool
	start    int64
	end      int64
}

func WithDownloadMaxBytes(maxBytes int64) DownloadOption {
	return func(o *downloadOptions) { o.maxBytes = maxBytes }
}

func WithDownloadRange(start, end int64) DownloadOption {
	return func(o *downloadOptions) {
		o.rangeSet = true
		o.start = start
		o.end = end
	}
}

func (c *Client) Download(ctx context.Context, url string, opts ...RequestOption) (io.ReadCloser, error) {
	req := NewRequest(url, http.MethodGet, opts...)
	if req.OptionError != nil {
		return nil, req.OptionError
	}
	if ctx == nil {
		ctx = context.Background()
	}
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

func (c *Client) DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	cfg := downloadOptions{maxBytes: c.downloadBytesLimit}
	for _, opt := range opts {
		opt(&cfg)
	}
	reqOpts := make([]RequestOption, 0, 1)
	if cfg.rangeSet {
		rangeValue := fmt.Sprintf("bytes=%d-", cfg.start)
		if cfg.end >= cfg.start {
			rangeValue = fmt.Sprintf("bytes=%d-%d", cfg.start, cfg.end)
		}
		reqOpts = append(reqOpts, WithHeader("Range", rangeValue))
	}
	body, err := c.Download(ctx, url, reqOpts...)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	limit := cfg.maxBytes
	if limit <= 0 {
		return io.ReadAll(body)
	}
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download body too large: limit %d bytes", limit)
	}
	return data, nil
}

func DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	return defaultClient.DownloadBytes(ctx, url, opts...)
}
