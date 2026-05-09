package netx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DefaultDownloadBytesLimit 默认下载最大字节数（32MB）。
const DefaultDownloadBytesLimit = 32 << 20

// DownloadOption 函数式下载配置选项。
type DownloadOption func(*downloadOptions)

type downloadOptions struct {
	maxBytes int64
	rangeSet bool
	start    int64
	end      int64
}

// WithDownloadMaxBytes 设置下载最大字节数限制。
func WithDownloadMaxBytes(maxBytes int64) DownloadOption {
	return func(o *downloadOptions) { o.maxBytes = maxBytes }
}

// WithDownloadRange 设置下载 Range 请求头。
func WithDownloadRange(start, end int64) DownloadOption {
	return func(o *downloadOptions) {
		o.rangeSet = true
		o.start = start
		o.end = end
	}
}

// Download 下载 URL 内容，返回 io.ReadCloser。调用方需自行关闭。
func (c *Client) Download(ctx context.Context, url string, opts ...DownloadOption) (io.ReadCloser, error) {
	dl := resolveDownloadOptions(opts)
	reqOpts := dl.requestOptions()
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req := NewRequest(url, http.MethodGet, reqOpts...)
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

// DownloadFile 下载 URL 内容并保存到本地文件。
func (c *Client) DownloadFile(ctx context.Context, url, destPath string, opts ...DownloadOption) error {
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

// DownloadBytes 下载 URL 内容并返回字节数据。
func (c *Client) DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	dl := downloadOptions{maxBytes: c.downloadBytesLimit}
	for _, opt := range opts {
		opt(&dl)
	}
	body, err := c.Download(ctx, url, opts...)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := readLimitedBody(body, dl.maxBytes)
	if err != nil {
		return nil, fmt.Errorf("download body too large: limit %d bytes", dl.maxBytes)
	}
	return data, nil
}

// DownloadBytes 使用默认 Client 下载 URL 内容并返回字节数据（包级别便捷函数）。
func DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	return defaultClient.DownloadBytes(ctx, url, opts...)
}

func resolveDownloadOptions(opts []DownloadOption) downloadOptions {
	var dl downloadOptions
	for _, opt := range opts {
		opt(&dl)
	}
	return dl
}

func (dl downloadOptions) requestOptions() []RequestOption {
	if !dl.rangeSet {
		return nil
	}
	rangeValue := fmt.Sprintf("bytes=%d-", dl.start)
	if dl.end >= dl.start {
		rangeValue = fmt.Sprintf("bytes=%d-%d", dl.start, dl.end)
	}
	return []RequestOption{WithHeader("Range", rangeValue)}
}
