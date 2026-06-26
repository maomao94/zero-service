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

// downloadRaw 发送下载请求，返回原始响应体 reader（不应用大小限制）。
func (c *Client) downloadRaw(ctx context.Context, url string, reqOpts ...RequestOption) (io.ReadCloser, error) {
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

// Download 下载 URL 内容，返回 io.ReadCloser。调用方需自行关闭。
// 受 client 级 downloadBytesLimit 及 WithDownloadMaxBytes 选项限制。
func (c *Client) Download(ctx context.Context, url string, opts ...DownloadOption) (io.ReadCloser, error) {
	dl := downloadOptions{maxBytes: c.downloadBytesLimit}
	for _, opt := range opts {
		opt(&dl)
	}
	body, err := c.downloadRaw(ctx, url, dl.requestOptions()...)
	if err != nil {
		return nil, err
	}
	if dl.maxBytes > 0 {
		body = http.MaxBytesReader(nil, body, dl.maxBytes)
	}
	return body, nil
}

// DownloadFile 下载 URL 内容并保存到本地文件（原子写入：先写临时文件，成功后再 rename）。
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
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	if _, err = io.Copy(f, body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}
	if err = f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close file: %w", err)
	}
	if err = os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename file: %w", err)
	}
	return nil
}

// DownloadBytes 下载 URL 内容并返回字节数据。超过限制时返回错误。
func (c *Client) DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	body, err := c.Download(ctx, url, opts...)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

// DownloadBytes 使用默认 Client 下载 URL 内容并返回字节数据（包级别便捷函数）。
func DownloadBytes(ctx context.Context, url string, opts ...DownloadOption) ([]byte, error) {
	return defaultClient.DownloadBytes(ctx, url, opts...)
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
