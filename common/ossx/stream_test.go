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
func (f *streamFakeTemplate) PutStream(context.Context, string, string, string, string, io.Reader, int64) (*File, error) {
	return nil, nil
}
func (f *streamFakeTemplate) PutObject(_ context.Context, tenantID, bucketName, filename, contentType string, reader io.Reader, objectSize int64, pathPrefix ...string) (*File, error) {
	f.contentType = contentType
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
