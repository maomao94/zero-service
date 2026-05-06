package filex

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadCaptureWriterKeepsOnlyConfiguredBytes(t *testing.T) {
	writer := NewHeadCaptureWriter(5)

	n, err := writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len("hello world") {
		t.Fatalf("Write() n = %d, want %d", n, len("hello world"))
	}
	if got := string(writer.Bytes()); got != "hello" {
		t.Fatalf("Bytes() = %q, want hello", got)
	}

	_, _ = writer.Write([]byte("again"))
	if got := string(writer.Bytes()); got != "hello" {
		t.Fatalf("Bytes() after second write = %q, want hello", got)
	}
}

func TestCaptureCollectsHeadAndOptionalTempFile(t *testing.T) {
	tempDir := t.TempDir()
	capture, err := NewCapture(CaptureOptions{
		TempDir:     tempDir,
		TempPattern: "upload-*",
		NeedTemp:    true,
		MaxHeadRead: 5,
	})
	if err != nil {
		t.Fatalf("NewCapture() error = %v", err)
	}

	for _, w := range capture.Writers() {
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Fatalf("writer.Write() error = %v", err)
		}
	}
	if string(capture.Head()) != "hello" {
		t.Fatalf("Head() = %q, want hello", string(capture.Head()))
	}
	if !capture.HasTempFile() {
		t.Fatal("HasTempFile() = false, want true")
	}

	if err := capture.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content, err := os.ReadFile(capture.TempFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("temp file content = %q, want hello world", string(content))
	}

	tempPath := capture.TempFilePath()
	if err := capture.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatal("temp file still exists after Release()")
	}
}

func TestCaptureCloseIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	capture, err := NewCapture(CaptureOptions{
		TempDir:  tempDir,
		NeedTemp: true,
	})
	if err != nil {
		t.Fatalf("NewCapture() error = %v", err)
	}
	if err := capture.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := capture.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if err := capture.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}

func TestCaptureNoTempFile(t *testing.T) {
	capture, err := NewCapture(CaptureOptions{MaxHeadRead: 5})
	if err != nil {
		t.Fatalf("NewCapture() error = %v", err)
	}
	if capture.HasTempFile() {
		t.Fatal("HasTempFile() = true, want false")
	}
	if capture.TempFilePath() != "" {
		t.Fatalf("TempFilePath() = %q, want empty", capture.TempFilePath())
	}
	if err := capture.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := capture.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}

func TestReadFileHead(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "probe.bin")
	content := []byte("hello-world-payload")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	head, size, err := ReadFileHead(path, 5)
	if err != nil {
		t.Fatalf("ReadFileHead() error = %v", err)
	}
	if size != int64(len(content)) {
		t.Fatalf("size = %d, want %d", size, len(content))
	}
	if string(head) != "hello" {
		t.Fatalf("head = %q, want hello", string(head))
	}

	head0, size0, err := ReadFileHead(path, 0)
	if err != nil {
		t.Fatalf("ReadFileHead maxHead=0 error = %v", err)
	}
	if size0 != int64(len(content)) {
		t.Fatalf("size0 = %d, want %d", size0, len(content))
	}
	if head0 != nil {
		t.Fatalf("head0 = %v, want nil", head0)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src.txt")
	dst := filepath.Join(tempDir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("copied content = %q, want hello", string(content))
	}
}

func TestNewMD5TeeReader(t *testing.T) {
	reader, digest := NewMD5TeeReader(strings.NewReader("hello world"))
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("data = %q, want hello world", string(data))
	}
	if digest.Hex() != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("Hex() = %q, want 5eb63bbbe01eeed093cb22bb8f5acdc3", digest.Hex())
	}
}
