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
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamMultipart(writer, multipartWriter, files, fields, c.uploadBytesLimit)
	}()
	req := NewRequest(url, http.MethodPost, opts...)
	req.BodyReader = reader
	req.Body = nil
	req.bodyKind = bodyKindReader
	req.ContentType = multipartWriter.FormDataContentType()
	resp, err := c.Do(ctx, req)
	if err != nil {
		_ = reader.CloseWithError(err)
		return nil, err
	}
	if streamErr := <-errCh; streamErr != nil && resp.Err == nil {
		resp.Success = false
		resp.StatusCode = http.StatusBadGateway
		resp.Err = streamErr
	}
	return resp, nil
}

func streamMultipart(pipeWriter *io.PipeWriter, multipartWriter *multipart.Writer, files []FileUpload, fields map[string]string, maxBytes int64) error {
	var err error
	defer func() {
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		_ = pipeWriter.Close()
	}()
	var written int64
	for _, f := range files {
		part, partErr := multipartWriter.CreateFormFile(f.FieldName, f.FileName)
		if partErr != nil {
			err = fmt.Errorf("create form file: %w", partErr)
			return err
		}
		writer := io.Writer(part)
		if maxBytes > 0 {
			remaining := maxBytes - written
			if remaining <= 0 {
				err = fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, maxBytes)
				return err
			}
			writer = &limitedWriter{w: part, n: remaining, limit: maxBytes}
		}
		n, partErr := io.Copy(writer, f.Content)
		written += n
		if partErr != nil {
			err = fmt.Errorf("copy file content: %w", partErr)
			return err
		}
	}
	for k, v := range fields {
		if fieldErr := multipartWriter.WriteField(k, v); fieldErr != nil {
			err = fmt.Errorf("write field: %w", fieldErr)
			return err
		}
	}
	if closeErr := multipartWriter.Close(); closeErr != nil {
		err = fmt.Errorf("close multipart writer: %w", closeErr)
		return err
	}
	return nil
}

type limitedWriter struct {
	w     io.Writer
	n     int64
	limit int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, w.limit)
	}
	if int64(len(p)) > w.n {
		n, err := w.w.Write(p[:w.n])
		if err != nil {
			return n, err
		}
		w.n -= int64(n)
		return n, fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, w.limit)
	}
	n, err := w.w.Write(p)
	w.n -= int64(n)
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
