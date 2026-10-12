package ossx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestTemplateRefreshesOnUpdateTimeChange(t *testing.T) {
	CacheInvalidate("t1") // 确保干净起点

	updateTime := time.Now()
	getConfig := func(ctx context.Context, tenantId, code string) (*Config, error) {
		return &Config{Category: Category_Minio, Endpoint: "http://m.example.com:9000",
			AccessKey: "ak", SecretKey: "sk", BucketName: "b", UpdateTime: updateTime}, nil
	}

	first, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("first Template err = %v", err)
	}
	// update_time 未变化 → 复用同一缓存客户端
	second, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("second Template err = %v", err)
	}
	if first != second {
		t.Fatal("update_time unchanged: want cached template reused")
	}
	// update_time 前进（配置被编辑）→ 重建客户端
	updateTime = updateTime.Add(time.Second)
	third, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("third Template err = %v", err)
	}
	if third == first {
		t.Fatal("update_time changed: want template rebuilt")
	}
}

func TestTemplateSeparatesByTenantAndCode(t *testing.T) {
	CacheInvalidate("t1")
	getConfig := func(_ context.Context, tenantId, code string) (*Config, error) {
		return &Config{Category: Category_Minio, Endpoint: "http://m.example.com:9000",
			AccessKey: "ak", SecretKey: "sk", BucketName: code}, nil
	}

	a, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("c1 Template err = %v", err)
	}
	b, err := Template(context.Background(), "t1", "c2", false, getConfig)
	if err != nil {
		t.Fatalf("c2 Template err = %v", err)
	}
	if a == b {
		t.Fatal("different oss_code must map to different cached templates")
	}
}

func TestTemplateInvalidateForcesReload(t *testing.T) {
	CacheInvalidate("t1")
	getConfig := func(_ context.Context, tenantId, code string) (*Config, error) {
		return &Config{Category: Category_Minio, Endpoint: "http://m.example.com:9000",
			AccessKey: "ak", SecretKey: "sk", BucketName: "b"}, nil
	}

	first, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("first Template err = %v", err)
	}
	CacheInvalidate("t1")
	second, err := Template(context.Background(), "t1", "c1", false, getConfig)
	if err != nil {
		t.Fatalf("after invalidate Template err = %v", err)
	}
	if second == first {
		t.Fatal("after CacheInvalidate: want template rebuilt")
	}
}

func TestBuildFileKeepsMd5(t *testing.T) {
	template := MinioTemplate{
		ossProperties: OssProperties{Endpoint: "http://oss"},
		ossRule:       OssRule{tenantMode: true},
	}
	got := template.buildFile("t1", "bucket", "upload/a.txt", "a.txt", 11, "5eb63bbbe01eeed093cb22bb8f5acdc3")
	if got.Md5 != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("Md5 = %q, want expected hash", got.Md5)
	}
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
