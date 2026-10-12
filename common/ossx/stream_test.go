package ossx

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"testing"
	"time"

	"zero-service/common/filex"
)

type streamFakeTemplate struct {
	data        bytes.Buffer
	contentType string
	size        int64
}

func (f *streamFakeTemplate) MakeBucket(context.Context, string, string) error   { return nil }
func (f *streamFakeTemplate) RemoveBucket(context.Context, string, string) error { return nil }
func (f *streamFakeTemplate) StatFile(context.Context, string, string, string) (*OssFile, error) {
	return nil, nil
}
func (f *streamFakeTemplate) BucketExists(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f *streamFakeTemplate) PutFile(context.Context, string, string, *multipart.FileHeader, ...string) (*File, error) {
	return nil, nil
}
func (f *streamFakeTemplate) PutObject(_ context.Context, tenantID, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) {
	f.contentType = contentType
	f.size = size
	_, _ = io.Copy(&f.data, reader)
	return &File{Name: filename, Size: int64(f.data.Len())}, nil
}
func (f *streamFakeTemplate) SignUrl(context.Context, string, string, string, time.Duration) (string, error) {
	return "", nil
}
func (f *streamFakeTemplate) RemoveFile(context.Context, string, string, string) error { return nil }
func (f *streamFakeTemplate) RemoveFiles(context.Context, string, string, []string) ([]RemoveFileResult, error) {
	return nil, nil
}

func TestUploadStreamDetectsContentTypeAndCapturesHead(t *testing.T) {
	template := &streamFakeTemplate{}
	result, err := UploadStream(context.Background(), StreamUploadRequest{
		Template:   template,
		TenantID:   "t1",
		BucketName: "bucket",
		Filename:   "a.png",
		Reader:     bytes.NewReader([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'x'}),
		Size:       9,
		CaptureOptions: filex.CaptureOptions{
			MaxHeadRead: 4,
		},
	})
	if err != nil {
		t.Fatalf("UploadStream() error = %v", err)
	}
	if result.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", result.ContentType)
	}
	if template.contentType != "image/png" {
		t.Fatalf("template contentType = %q, want image/png", template.contentType)
	}
	if template.data.Len() != 9 {
		t.Fatalf("uploaded size = %d, want 9", template.data.Len())
	}
	if got := string(result.Head); got != "\x89PNG" {
		t.Fatalf("head = %q, want png signature prefix", got)
	}
}

// TestUploadStreamUnknownSizeUsesProviderSize：未知大小上传成功时，结果大小取 provider 回填值。
func TestUploadStreamUnknownSizeUsesProviderSize(t *testing.T) {
	result, err := UploadStream(context.Background(), StreamUploadRequest{
		Template:   &streamFakeTemplate{},
		TenantID:   "t1",
		BucketName: "bucket",
		Filename:   "a.bin",
		Reader:     bytes.NewReader([]byte("hello")),
		Size:       -1,
	})
	if err != nil {
		t.Fatalf("UploadStream() error = %v", err)
	}
	if result.Size != 5 {
		t.Fatalf("Size = %d, want 5", result.Size)
	}
}

// TestUploadStreamUnknownSizeEmptyFallsBackToZero：未知大小上传空流时归零，不把 -1 透出到响应。
func TestUploadStreamUnknownSizeEmptyFallsBackToZero(t *testing.T) {
	result, err := UploadStream(context.Background(), StreamUploadRequest{
		Template:   &streamFakeTemplate{},
		TenantID:   "t1",
		BucketName: "bucket",
		Filename:   "empty.bin",
		Reader:     bytes.NewReader(nil),
		Size:       -1,
	})
	if err != nil {
		t.Fatalf("UploadStream() error = %v", err)
	}
	if result.Size != 0 {
		t.Fatalf("Size = %d, want 0", result.Size)
	}
}

// TestUploadStreamNormalizesNonPositiveSizeToUnknown：0（proto 默认/未设置）与负值统一归一为未知大小(-1)，
// provider 不应再收到 0，避免被当作已知空文件丢数据。
func TestUploadStreamNormalizesNonPositiveSizeToUnknown(t *testing.T) {
	template := &streamFakeTemplate{}
	result, err := UploadStream(context.Background(), StreamUploadRequest{
		Template:   template,
		TenantID:   "t1",
		BucketName: "bucket",
		Filename:   "a.bin",
		Reader:     bytes.NewReader([]byte("hello")),
		Size:       0,
	})
	if err != nil {
		t.Fatalf("UploadStream() error = %v", err)
	}
	if template.size != -1 {
		t.Fatalf("provider size = %d, want -1", template.size)
	}
	if result.Size != 5 {
		t.Fatalf("Size = %d, want 5", result.Size)
	}
}
