package ossx

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func TestOssRuleFullBucketName(t *testing.T) {
	tests := []struct {
		name       string
		tenantMode bool
		tenantID   string
		bucketName string
		want       string
	}{
		{name: "tenant mode disabled", tenantMode: false, tenantID: "t1", bucketName: "bucket", want: "bucket"},
		{name: "tenant mode enabled", tenantMode: true, tenantID: "t1", bucketName: "bucket", want: "t1-bucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := OssRule{tenantMode: tt.tenantMode}
			got := rule.fullBucketName(tt.tenantID, tt.bucketName)
			if got != tt.want {
				t.Errorf("fullBucketName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOssRuleFilename(t *testing.T) {
	rule := OssRule{}

	got := rule.filename("photo.jpg", "avatar")
	if !strings.HasPrefix(got, "avatar/") {
		t.Fatalf("filename() = %q, want avatar prefix", got)
	}
	if !strings.HasSuffix(got, ".jpg") {
		t.Fatalf("filename() = %q, want .jpg suffix", got)
	}
	parts := strings.Split(got, "/")
	if len(parts) != 3 {
		t.Fatalf("filename() = %q, want prefix/date/name.ext", got)
	}
	if len(parts[1]) != 8 {
		t.Fatalf("filename() date segment = %q, want 8 characters", parts[1])
	}
}

func TestOssRuleFilenameDefaultPrefix(t *testing.T) {
	rule := OssRule{}
	got := rule.filename("archive.tar.gz")
	if !strings.HasPrefix(got, "upload/") {
		t.Fatalf("filename() = %q, want upload prefix", got)
	}
	if !strings.HasSuffix(got, ".gz") {
		t.Fatalf("filename() = %q, want .gz suffix", got)
	}
}

func TestOssRuleFilenameFixedObjectName(t *testing.T) {
	rule := OssRule{}
	got := rule.filename("photo.jpg", "", "thumb/20260507/fixed.jpg")
	want := "thumb/20260507/fixed.jpg"
	if got != want {
		t.Errorf("filename() = %q, want %q", got, want)
	}
}

func TestNeedRebuild(t *testing.T) {
	cached := &Config{Endpoint: "e1", AccessKey: "a1", SecretKey: "s1"}
	current := &Config{Endpoint: "e1", AccessKey: "a1", SecretKey: "s1"}
	if needRebuild(cached, fakeOssTemplate{}, current) {
		t.Fatal("needRebuild() = true, want false for same credentials")
	}

	current.SecretKey = "s2"
	if !needRebuild(cached, fakeOssTemplate{}, current) {
		t.Fatal("needRebuild() = false, want true after secret key changed")
	}
	if !needRebuild(nil, fakeOssTemplate{}, current) {
		t.Fatal("needRebuild() = false, want true when cached config is nil")
	}
	if !needRebuild(cached, nil, current) {
		t.Fatal("needRebuild() = false, want true when template is nil")
	}
}

func TestBuildFileKeepsMd5(t *testing.T) {
	template := MinioTemplate{
		ossProperties: OssProperties{Endpoint: "http://oss"},
		ossRule:       OssRule{tenantMode: true},
	}
	got := template.buildFile("t1", "bucket", "upload/a.txt", "a.txt", minio.UploadInfo{Size: 11}, "5eb63bbbe01eeed093cb22bb8f5acdc3")
	if got.Md5 != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("Md5 = %q, want expected hash", got.Md5)
	}
}

type fakeOssTemplate struct{}

func (fakeOssTemplate) MakeBucket(context.Context, string, string) error   { return nil }
func (fakeOssTemplate) RemoveBucket(context.Context, string, string) error { return nil }
func (fakeOssTemplate) StatFile(context.Context, string, string, string) (*OssFile, error) {
	return nil, nil
}
func (fakeOssTemplate) BucketExists(context.Context, string, string) (bool, error) { return true, nil }
func (fakeOssTemplate) PutFile(context.Context, string, string, *multipart.FileHeader, ...string) (*File, error) {
	return nil, nil
}
func (fakeOssTemplate) PutStream(context.Context, string, string, string, string, io.Reader, int64) (*File, error) {
	return nil, nil
}
func (fakeOssTemplate) PutObject(context.Context, string, string, string, string, io.Reader, int64, ...string) (*File, error) {
	return nil, nil
}
func (fakeOssTemplate) SignUrl(context.Context, string, string, string, time.Duration) (string, error) {
	return "", nil
}
func (fakeOssTemplate) RemoveFile(context.Context, string, string, string) error { return nil }
func (fakeOssTemplate) RemoveFiles(context.Context, string, string, []string) ([]RemoveFileResult, error) {
	return nil, nil
}

type templateCtxKey struct{}

func TestTemplatePassesContextToGetConfig(t *testing.T) {
	ctx := context.WithValue(context.Background(), templateCtxKey{}, "marker")
	_, err := Template(ctx, "tid", "code", false, func(c context.Context, tenantID, code string) (*Config, error) {
		if c.Value(templateCtxKey{}) != "marker" {
			t.Error("GetConfigFn received wrong context")
		}
		return nil, errors.New("abort")
	})
	if err == nil || err.Error() != "abort" {
		t.Fatalf("Template err = %v, want abort", err)
	}
}
