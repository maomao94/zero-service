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

	var baseWriter io.Writer = writer
	var counter *countingWriter
	if c.uploadBytesLimit > 0 {
		counter = &countingWriter{w: writer, limit: c.uploadBytesLimit}
		baseWriter = counter
	}
	multipartWriter := multipart.NewWriter(baseWriter)

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamMultipart(writer, multipartWriter, files, fields, counter)
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
		resp.StatusCode = http.StatusBadGateway
		resp.Success = false
		resp.Err = streamErr
	}
	return resp, nil
}

func streamMultipart(pipeWriter *io.PipeWriter, multipartWriter *multipart.Writer, files []FileUpload, fields map[string]string, counter *countingWriter) error {
	var err error
	defer func() {
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		_ = pipeWriter.Close()
	}()

	for _, f := range files {
		part, partErr := multipartWriter.CreateFormFile(f.FieldName, f.FileName)
		if partErr != nil {
			err = fmt.Errorf("create form file: %w", partErr)
			return err
		}
		if _, partErr = io.Copy(part, f.Content); partErr != nil {
			err = fmt.Errorf("copy file content: %w", partErr)
			return err
		}
		if counter != nil && counter.exceeded {
			err = fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, counter.limit)
			return err
		}
	}
	for k, v := range fields {
		if fieldErr := multipartWriter.WriteField(k, v); fieldErr != nil {
			err = fmt.Errorf("write field: %w", fieldErr)
			return err
		}
		if counter != nil && counter.exceeded {
			err = fmt.Errorf("%w: limit %d bytes", ErrUploadTooLarge, counter.limit)
			return err
		}
	}
	if closeErr := multipartWriter.Close(); closeErr != nil {
		err = fmt.Errorf("close multipart writer: %w", closeErr)
		return err
	}
	return nil
}

type countingWriter struct {
	w        io.Writer
	written  int64
	limit    int64
	exceeded bool
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.written += int64(n)
	if cw.written > cw.limit {
		cw.exceeded = true
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
