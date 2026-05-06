package logic

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/threading"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/config"
	"zero-service/app/file/internal/svc"
	"zero-service/common/netx"
	"zero-service/common/ossx"
)

type logicFakeTemplate struct {
	data          bytes.Buffer
	contentType   string
	putCalls      int
	err           error
	skipReadOnErr bool
	ctxErr        error
}

func (t *logicFakeTemplate) MakeBucket(context.Context, string, string) error   { return nil }
func (t *logicFakeTemplate) RemoveBucket(context.Context, string, string) error { return nil }
func (t *logicFakeTemplate) StatFile(context.Context, string, string, string) (*ossx.OssFile, error) {
	return nil, nil
}
func (t *logicFakeTemplate) BucketExists(context.Context, string, string) (bool, error) {
	return true, nil
}
func (t *logicFakeTemplate) PutFile(context.Context, string, string, *multipart.FileHeader, ...string) (*ossx.File, error) {
	return nil, nil
}
func (t *logicFakeTemplate) PutStream(context.Context, string, string, string, string, io.Reader, int64) (*ossx.File, error) {
	return nil, nil
}
func (t *logicFakeTemplate) PutObject(ctx context.Context, tenantID, bucketName, filename, contentType string, reader io.Reader, objectSize int64, pathPrefix ...string) (*ossx.File, error) {
	t.putCalls++
	t.contentType = contentType
	t.ctxErr = ctx.Err()
	if t.ctxErr != nil {
		return nil, t.ctxErr
	}
	if t.err != nil && t.skipReadOnErr {
		return nil, t.err
	}
	_, _ = io.Copy(&t.data, reader)
	if t.err != nil {
		return nil, t.err
	}
	return &ossx.File{
		Name:   filename,
		Link:   "http://oss/" + filename,
		Domain: "http://oss",
		Size:   int64(t.data.Len()),
		Md5:    fmt.Sprintf("%x", md5.Sum(t.data.Bytes())),
	}, nil
}
func (t *logicFakeTemplate) SignUrl(context.Context, string, string, string, time.Duration) (string, error) {
	return "", nil
}
func (t *logicFakeTemplate) RemoveFile(context.Context, string, string, string) error { return nil }
func (t *logicFakeTemplate) RemoveFiles(context.Context, string, string, []string) ([]ossx.RemoveFileResult, error) {
	return nil, nil
}

// ======================== buildCaptureOptions 测试 ========================

func TestBuildCaptureOptions_ThumbDisabled(t *testing.T) {
	opts := buildCaptureOptions(config.UploadConf{
		TempDir: "/tmp/test",
		Image: config.ImageUploadConf{
			MaxExifRead: 65536,
		},
	}, false)
	if opts.TempDir != "/tmp/test" {
		t.Fatalf("TempDir = %q, want /tmp/test", opts.TempDir)
	}
	if opts.NeedTemp {
		t.Fatal("NeedTemp should be false when Thumb is disabled")
	}
	if opts.MaxHeadRead != 65536 {
		t.Fatalf("MaxHeadRead = %d, want 65536", opts.MaxHeadRead)
	}
}

func TestBuildCaptureOptions_ThumbEnabled(t *testing.T) {
	opts := buildCaptureOptions(config.UploadConf{
		TempDir: "/tmp/test",
		Image: config.ImageUploadConf{
			MaxExifRead: 65536,
			Thumb:       config.ImageVariantConf{Enabled: true, Width: 100, Height: 100},
		},
	}, false)
	if opts.NeedTemp {
		t.Fatal("NeedTemp should be false when thumb generation is not requested")
	}
	if opts.TempPattern != "upload-*" {
		t.Fatalf("TempPattern = %q, want upload-*", opts.TempPattern)
	}
}

func TestBuildCaptureOptions_IsThumb(t *testing.T) {
	opts := buildCaptureOptions(config.UploadConf{
		TempDir: "/tmp/test",
		Image: config.ImageUploadConf{
			Thumb: config.ImageVariantConf{Enabled: true, Width: 100, Height: 100},
		},
	}, true)
	if !opts.NeedTemp {
		t.Fatal("NeedTemp should be true when synchronous thumb generation is requested")
	}
}

func TestBuildCaptureOptions_ZeroWidthHeight(t *testing.T) {
	opts := buildCaptureOptions(config.UploadConf{
		TempDir: "/tmp/test",
		Image: config.ImageUploadConf{
			Thumb: config.ImageVariantConf{Enabled: true, Width: 0, Height: 100},
		},
	}, false)
	if opts.NeedTemp {
		t.Fatal("NeedTemp should be false when Thumb width is 0")
	}
}

// ======================== processUploadResult 测试 ========================

func TestProcessUploadResult_NonImage(t *testing.T) {
	tempDir := t.TempDir()
	result := &ossx.StreamUploadResult{
		File: &ossx.File{
			Name:   "a.txt",
			Link:   "http://oss/a.txt",
			Domain: "http://oss",
			Size:   11,
			Md5:    "abc123",
		},
		ContentType: "text/plain",
		Size:        11,
	}

	pbFile := processUploadResult(
		context.Background(),
		config.UploadConf{TempDir: tempDir},
		result, &logicFakeTemplate{},
		"t1", "bucket", "a.txt", false, nil,
	)

	if pbFile.Name != "a.txt" {
		t.Fatalf("Name = %q, want a.txt", pbFile.Name)
	}
	if pbFile.Md5 != "abc123" {
		t.Fatalf("Md5 = %q, want abc123", pbFile.Md5)
	}
	if pbFile.Meta != nil {
		t.Fatal("Meta should be nil for non-image")
	}
	// 非图片临时文件已清理
	ensureTempDirEmpty(t, tempDir)
}

func TestProcessUploadResult_ImageExtractsEXIF(t *testing.T) {
	tempDir := t.TempDir()
	pngHead := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'x', 'y'}
	result := &ossx.StreamUploadResult{
		File: &ossx.File{
			Name:   "a.png",
			Link:   "http://oss/a.png",
			Domain: "http://oss",
			Size:   10,
			Md5:    "def456",
		},
		ContentType: "image/png",
		Size:        10,
		Head:        pngHead,
	}

	pbFile := processUploadResult(
		context.Background(),
		config.UploadConf{TempDir: tempDir},
		result, &logicFakeTemplate{},
		"t1", "bucket", "a.png", false, nil,
	)

	if !isImageContentType(result.ContentType) {
		t.Fatal("PNG should be detected as image")
	}
	if pbFile.ThumbLink != "" || pbFile.ThumbName != "" {
		t.Fatal("Thumb fields should be empty when thumb is disabled")
	}
	ensureTempDirEmpty(t, tempDir)
}

func TestProcessUploadResult_GeneratesThumbWhenRequested(t *testing.T) {
	tempDir := t.TempDir()
	pngHead := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'x', 'y'}
	tempPath := newTestTempFile(t, tempDir, pngHead)
	result := &ossx.StreamUploadResult{
		File: &ossx.File{
			Name:   "a.png",
			Link:   "http://oss/a.png",
			Domain: "http://oss",
			Size:   10,
			Md5:    "def456",
		},
		ContentType: "image/png",
		Size:        10,
		Head:        pngHead,
		TempPath:    tempPath,
	}

	pbFile := processUploadResult(
		context.Background(),
		config.UploadConf{
			TempDir: tempDir,
			Image: config.ImageUploadConf{
				Thumb: config.ImageVariantConf{Enabled: true, Width: 100, Height: 100},
			},
		},
		result, &logicFakeTemplate{},
		"t1", "bucket", "a.png", true, nil,
	)

	if pbFile.ThumbLink != "" || pbFile.ThumbName != "" {
		t.Fatal("Thumb fields should be empty when async runner is missing")
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatal("source temp file should be removed when async runner is missing")
	}
	ensureTempDirEmpty(t, tempDir)
}

func TestProcessUploadResult_NoThumbWhenTempPathMissing(t *testing.T) {
	tempDir := t.TempDir()
	pngHead := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'x', 'y'}
	result := &ossx.StreamUploadResult{
		File: &ossx.File{
			Name:   "a.png",
			Link:   "http://oss/a.png",
			Domain: "http://oss",
			Size:   10,
			Md5:    "def456",
		},
		ContentType: "image/png",
		Size:        10,
		Head:        pngHead,
	}

	pbFile := processUploadResult(
		context.Background(),
		config.UploadConf{
			TempDir: tempDir,
			Image: config.ImageUploadConf{
				Thumb: config.ImageVariantConf{Enabled: true, Width: 100, Height: 100},
			},
		},
		result, &logicFakeTemplate{},
		"t1", "bucket", "a.png", true, nil,
	)

	if pbFile.ThumbLink != "" || pbFile.ThumbName != "" {
		t.Fatal("Thumb fields should be empty when temp source is missing")
	}
	ensureTempDirEmpty(t, tempDir)
}

func TestProcessUploadResult_AsyncThumbUsesContextWithoutCancel(t *testing.T) {
	tempDir := t.TempDir()
	pngHead := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'x', 'y'}
	template := &logicFakeTemplate{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := &ossx.StreamUploadResult{
		File: &ossx.File{
			Name:   "a.png",
			Link:   "http://oss/a.png",
			Domain: "http://oss",
			Size:   10,
			Md5:    "def456",
		},
		ContentType: "image/png",
		Size:        10,
		Head:        pngHead,
		TempPath:    newTestImageFile(t, tempDir),
	}

	pbFile := processUploadResult(
		ctx,
		config.UploadConf{
			TempDir: tempDir,
			Image: config.ImageUploadConf{
				Thumb: config.ImageVariantConf{Enabled: true, Width: 16, Height: 16},
			},
		},
		result, template,
		"t1", "bucket", "a.png", true, threading.NewTaskRunner(1),
	)

	if pbFile.ThumbLink == "" || pbFile.ThumbName == "" {
		t.Fatal("Thumb fields should be set before async task finishes")
	}
	if template.ctxErr != nil {
		t.Fatalf("async upload ctxErr = %v, want nil", template.ctxErr)
	}
	time.Sleep(20 * time.Millisecond)
}

func TestProcessUploadResult_NilFile(t *testing.T) {
	tempDir := t.TempDir()
	result := &ossx.StreamUploadResult{
		File:        nil,
		ContentType: "text/plain",
	}

	pbFile := processUploadResult(
		context.Background(),
		config.UploadConf{TempDir: tempDir},
		result, &logicFakeTemplate{},
		"t1", "bucket", "a.txt", false, nil,
	)

	if pbFile.Name != "" {
		t.Fatal("expected empty File for nil result.File")
	}
}

// ======================== 辅助函数 ========================

func newTestTempFile(t *testing.T, tempDir string, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp(tempDir, "upload-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		t.Fatalf("Write() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return f.Name()
}

func newTestImageFile(t *testing.T, tempDir string) string {
	t.Helper()
	f, err := os.CreateTemp(tempDir, "upload-*.png")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatalf("Encode() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return f.Name()
}

func ensureTempDirEmpty(t *testing.T, tempDir string) {
	t.Helper()
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dir entries = %d, want 0", len(entries))
	}
}

// newRelayLogicForTest 构造 RelayFileLogic：svcCtx 仅提供 Upload 配置，resolver 注入假 OSS。
func newRelayLogicForTest(t *testing.T, uploadConf config.UploadConf, resolver ossx.TemplateResolver) *RelayFileLogic {
	t.Helper()
	return NewRelayFileLogic(context.Background(), &svc.ServiceContext{
		Config:              config.Config{Upload: uploadConf},
		NetClient:           netx.NewClient(),
		OssTemplateResolver: resolver,
	})
}

// ======================== Relay 测试（独立于 helper 重构）=======================

func TestRelayFileCopiesDetectedHeadBytes(t *testing.T) {
	source := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte("x"), 600)...)
	tempFile, err := os.CreateTemp(t.TempDir(), "source-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if _, err := tempFile.Write(source); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	template := &logicFakeTemplate{}
	logic := newRelayLogicForTest(t, config.UploadConf{TempDir: t.TempDir()}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		return template, nil
	})
	_, err = logic.RelayFile(&file.RelayFileReq{
		SourcePath: tempFile.Name(),
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.png"},
		},
	})
	if err != nil {
		t.Fatalf("RelayFile() error = %v", err)
	}
	if !bytes.Equal(template.data.Bytes(), source) {
		t.Fatalf("relay data length = %d, want %d", template.data.Len(), len(source))
	}
	if template.contentType != "image/png" {
		t.Fatalf("contentType = %q, want image/png", template.contentType)
	}
}

func TestRelayFileCopiesSourceToAllTargets(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "source-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if _, err := tempFile.WriteString("relay data"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	templates := []*logicFakeTemplate{{}, {}}
	logic := newRelayLogicForTest(t, config.UploadConf{TempDir: t.TempDir()}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		if code == "one" {
			return templates[0], nil
		}
		return templates[1], nil
	})
	res, err := logic.RelayFile(&file.RelayFileReq{
		SourcePath: tempFile.Name(),
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.txt"},
			{TenantId: "t1", Code: "two", BucketName: "bucket", Filename: "b.txt"},
		},
	})
	if err != nil {
		t.Fatalf("RelayFile() error = %v", err)
	}
	if len(res.Files) != 2 {
		t.Fatalf("files length = %d, want 2", len(res.Files))
	}
	if res.Files[0].Md5 == "" || res.Files[1].Md5 == "" {
		t.Fatal("relay response md5 should not be empty")
	}
	for i, template := range templates {
		if template.putCalls != 1 {
			t.Fatalf("target %d put calls = %d, want 1", i, template.putCalls)
		}
		if template.data.String() != "relay data" {
			t.Fatalf("target %d data = %q, want relay data", i, template.data.String())
		}
	}
}

func TestRelayFileContinuesAfterOneTargetPutObjectFails(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "source-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if _, err := tempFile.WriteString("relay data"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	putErr := errors.New("put failed")
	templates := []*logicFakeTemplate{{err: putErr, skipReadOnErr: true}, {}}
	logic := newRelayLogicForTest(t, config.UploadConf{TempDir: t.TempDir()}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		if code == "one" {
			return templates[0], nil
		}
		return templates[1], nil
	})
	res, err := logic.RelayFile(&file.RelayFileReq{
		SourcePath: tempFile.Name(),
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.txt"},
			{TenantId: "t1", Code: "two", BucketName: "bucket", Filename: "b.txt"},
		},
	})
	if !errors.Is(err, putErr) {
		t.Fatalf("RelayFile() error = %v, want %v", err, putErr)
	}
	if len(res.Files) != 1 {
		t.Fatalf("files length = %d, want 1", len(res.Files))
	}
	if templates[0].data.Len() != 0 {
		t.Fatalf("failed target data length = %d, want 0", templates[0].data.Len())
	}
	if templates[1].putCalls != 1 {
		t.Fatalf("second target put calls = %d, want 1", templates[1].putCalls)
	}
	if templates[1].data.String() != "relay data" {
		t.Fatalf("second target data = %q, want relay data", templates[1].data.String())
	}
}

func TestRelayFileFromURL(t *testing.T) {
	source := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte("x"), 600)...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(source)
	}))
	defer server.Close()

	template := &logicFakeTemplate{}
	logic := newRelayLogicForTest(t, config.UploadConf{TempDir: t.TempDir()}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		return template, nil
	})
	_, err := logic.RelayFile(&file.RelayFileReq{
		SourceUrl: server.URL + "/test.png",
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.png"},
		},
	})
	if err != nil {
		t.Fatalf("RelayFile() error = %v", err)
	}
	if !bytes.Equal(template.data.Bytes(), source) {
		t.Fatalf("relay data length = %d, want %d", template.data.Len(), len(source))
	}
	if template.contentType != "image/png" {
		t.Fatalf("contentType = %q, want image/png", template.contentType)
	}
}

func TestRelayFileFromURLNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logic := newRelayLogicForTest(t, config.UploadConf{TempDir: t.TempDir()}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		return &logicFakeTemplate{}, nil
	})
	_, err := logic.RelayFile(&file.RelayFileReq{
		SourceUrl: server.URL + "/missing",
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.txt"},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestRelayFileNoSource(t *testing.T) {
	logic := newRelayLogicForTest(t, config.UploadConf{}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		return &logicFakeTemplate{}, nil
	})
	_, err := logic.RelayFile(&file.RelayFileReq{
		Targets: []*file.RelayTarget{
			{TenantId: "t1", Code: "one", BucketName: "bucket", Filename: "a.txt"},
		},
	})
	if err == nil {
		t.Fatal("expected error when no source provided")
	}
}

func TestRelayFileNoTargets(t *testing.T) {
	logic := newRelayLogicForTest(t, config.UploadConf{}, func(ctx context.Context, tenantID, code string) (ossx.OssTemplate, error) {
		return &logicFakeTemplate{}, nil
	})
	_, err := logic.RelayFile(&file.RelayFileReq{
		SourcePath: "/dummy",
	})
	if err == nil {
		t.Fatal("expected error when no targets provided")
	}
}
