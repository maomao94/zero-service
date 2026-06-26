package netx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// Upload 流式上传文件到指定 URL，支持同时发送额外表单字段。
func (c *Client) Upload(ctx context.Context, url string, files []FileUpload, fields map[string]string, opts ...RequestOption) (*Response, error) {
	pr, pw := io.Pipe()

	w := io.Writer(pw)
	if c.uploadBytesLimit > 0 {
		w = &countingWriter{w: pw, limit: c.uploadBytesLimit}
	}
	mw := multipart.NewWriter(w)

	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in upload stream: %v", r)
			}
		}()
		errCh <- writeMultipart(pw, mw, files, fields)
	}()
	req := NewRequest(url, http.MethodPost, opts...)
	req.BodyReader = pr
	req.Body = nil
	req.FormData = nil
	req.bodyKind = bodyKindReader
	req.ContentType = mw.FormDataContentType()
	resp, err := c.Do(ctx, req)
	if err != nil {
		_ = pr.CloseWithError(err)
		return nil, err
	}
	if streamErr := <-errCh; streamErr != nil && resp.Err == nil {
		resp.StatusCode = http.StatusBadGateway
		resp.Success = false
		resp.Err = streamErr
	}
	return resp, nil
}

func writeMultipart(pw *io.PipeWriter, mw *multipart.Writer, files []FileUpload, fields map[string]string) (err error) {
	defer func() {
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()

	for k, v := range fields {
		if err = mw.WriteField(k, v); err != nil {
			return fmt.Errorf("write field: %w", err)
		}
	}
	for _, f := range files {
		if f.Content == nil {
			return fmt.Errorf("file content is nil for field %q", f.FieldName)
		}
		part, err := mw.CreateFormFile(f.FieldName, f.FileName)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err = io.Copy(part, f.Content); err != nil {
			return fmt.Errorf("copy file content: %w", err)
		}
	}
	if err = mw.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}
	return nil
}

type countingWriter struct {
	w       io.Writer
	written int64
	limit   int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.written += int64(n)
	if cw.written > cw.limit {
		return n, fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, cw.limit)
	}
	return n, err
}

// UploadFile 上传本地文件到指定 URL。
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

// UploadBytes 将内存数据作为文件上传到指定 URL。
func (c *Client) UploadBytes(ctx context.Context, url, fieldName, fileName string, data []byte, fields map[string]string, opts ...RequestOption) (*Response, error) {
	return c.Upload(ctx, url, []FileUpload{
		{FieldName: fieldName, FileName: fileName, Content: bytes.NewReader(data)},
	}, fields, opts...)
}
