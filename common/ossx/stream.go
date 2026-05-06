package ossx

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"zero-service/common/filex"
)

const maxContentTypeDetectBytes = 512

// StreamUploadRequest 描述一次通用流式上传请求。
type StreamUploadRequest struct {
	Template       OssTemplate
	TenantID       string
	BucketName     string
	Filename       string
	ContentType    string
	Reader         io.Reader
	Size           int64
	PathPrefix     string
	CaptureOptions filex.CaptureOptions
}

// StreamUploadResult 保存通用流上传结果与捕获到的简单后处理数据。
type StreamUploadResult struct {
	File        *File
	Size        int64
	ContentType string
	Head        []byte
	TempPath    string
}

// UploadStream 执行通用流上传并返回捕获结果，方便业务层做后处理。
func UploadStream(ctx context.Context, req StreamUploadRequest) (*StreamUploadResult, error) {
	head, reader, err := ReadUploadHead(req.Reader)
	if err != nil {
		return nil, err
	}

	contentType := DetectContentType(req.ContentType, head)
	capture, err := filex.NewCapture(req.CaptureOptions)
	if err != nil {
		return nil, err
	}

	uploadedFile, err := req.Template.PutObject(
		ctx,
		req.TenantID,
		req.BucketName,
		req.Filename,
		contentType,
		io.TeeReader(reader, io.MultiWriter(capture.Writers()...)),
		req.Size,
		req.PathPrefix,
	)
	if err != nil {
		_ = capture.Release()
		return nil, err
	}

	_ = capture.Close()

	written := req.Size
	if uploadedFile != nil && uploadedFile.Size > 0 {
		written = uploadedFile.Size
	}
	return &StreamUploadResult{
		File:        uploadedFile,
		Size:        written,
		ContentType: contentType,
		Head:        capture.Head(),
		TempPath:    capture.TempFilePath(),
	}, nil
}

// DetectContentType 在未显式指定时基于文件头探测 MIME 类型。
func DetectContentType(contentType string, head []byte) string {
	if contentType != "" {
		return contentType
	}
	if len(head) == 0 {
		return ""
	}
	return http.DetectContentType(head[:min(len(head), maxContentTypeDetectBytes)])
}

// ReadUploadHead 从 reader 中读取前 512 字节作为 MIME 类型与 EXIF 的探测头。
// 返回 head 切片和一个新的 io.Reader——该 reader 会将 head 与剩余数据合流，确保下游能读取完整内容。
func ReadUploadHead(reader io.Reader) ([]byte, io.Reader, error) {
	if reader == nil {
		return nil, nil, io.EOF
	}
	buf := make([]byte, maxContentTypeDetectBytes)
	n, err := io.ReadFull(reader, buf)
	if err == io.EOF {
		return nil, reader, nil
	}
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, nil, err
	}
	head := buf[:n]
	return head, io.MultiReader(bytes.NewReader(head), reader), nil
}
