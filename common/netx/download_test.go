package netx

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_Download(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("download content"))
	}))
	defer ts.Close()

	c := NewClient()
	body, err := c.Download(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "download content" {
		t.Errorf("expected download content, got %q", string(data))
	}
}

func TestClient_DownloadBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("byte content"))
	}))
	defer ts.Close()

	c := NewClient()
	data, err := c.DownloadBytes(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "byte content" {
		t.Errorf("expected byte content, got %q", string(data))
	}
}

func TestClient_DownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("saved content"))
	}))
	defer ts.Close()

	destPath := filepath.Join(t.TempDir(), "sub", "output.txt")
	c := NewClient()
	err := c.DownloadFile(ctx(t), ts.URL, destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "saved content" {
		t.Errorf("expected saved content, got %q", string(data))
	}
}

func TestClient_Download_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewClient()
	_, err := c.Download(ctx(t), ts.URL)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_DownloadBytes_DefaultLimitAndCustomLimit(t *testing.T) {
	large := strings.Repeat("a", DefaultDownloadBytesLimit+1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(large)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(large))
	}))
	defer ts.Close()

	c := NewClient()
	_, err := c.DownloadBytes(ctx(t), ts.URL)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected default limit error, got %v", err)
	}
	data, err := c.DownloadBytes(ctx(t), ts.URL, WithDownloadMaxBytes(int64(len(large))))
	if err != nil {
		t.Fatalf("unexpected custom limit error: %v", err)
	}
	if string(data) != large {
		t.Fatalf("unexpected custom limited data length %d", len(data))
	}
}

func TestClient_DownloadBytes_UsesClientConfiguredLimit(t *testing.T) {
	body := "abcdef"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer ts.Close()

	c := NewClient(WithDownloadBytesLimit(3))
	_, err := c.DownloadBytes(ctx(t), ts.URL)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected configured limit error, got %v", err)
	}
	data, err := c.DownloadBytes(ctx(t), ts.URL, WithDownloadMaxBytes(0))
	if err != nil {
		t.Fatalf("unexpected disabled limit error: %v", err)
	}
	if string(data) != body {
		t.Fatalf("expected full body, got %q", string(data))
	}
}

func TestClient_DownloadFile_RespectsMaxBytes(t *testing.T) {
	large := strings.Repeat("x", 1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(large))
	}))
	defer ts.Close()

	destPath := filepath.Join(t.TempDir(), "output.bin")
	c := NewClient()
	err := c.DownloadFile(ctx(t), ts.URL, destPath, WithDownloadMaxBytes(100))
	if err == nil {
		t.Fatal("expected error for oversized download")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
	if _, statErr := os.Stat(destPath); statErr == nil {
		t.Fatal("destination file should not exist on overflow")
	}
}

func TestClient_Download_RespectsMaxBytes(t *testing.T) {
	large := strings.Repeat("x", 1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(large))
	}))
	defer ts.Close()

	c := NewClient()
	body, err := c.Download(ctx(t), ts.URL, WithDownloadMaxBytes(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	_, err = io.ReadAll(body)
	if err == nil {
		t.Fatal("expected overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestClient_Download_ExactMaxBytes_NoError(t *testing.T) {
	exact := strings.Repeat("x", 100)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exact))
	}))
	defer ts.Close()

	c := NewClient()
	body, err := c.Download(ctx(t), ts.URL, WithDownloadMaxBytes(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("expected no error for exact size, got %v", err)
	}
	if string(data) != exact {
		t.Fatalf("expected full data, got %d bytes", len(data))
	}
}

func TestClient_DownloadBytes_Range(t *testing.T) {
	var gotRange string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("hello"))
	}))
	defer ts.Close()

	data, err := NewClient().DownloadBytes(ctx(t), ts.URL, WithDownloadRange(0, 4))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected range data, got %q", string(data))
	}
	if gotRange != "bytes=0-4" {
		t.Fatalf("expected range header, got %q", gotRange)
	}
}

func TestClient_Download_RangeStartOnly(t *testing.T) {
	var gotRange string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("tail"))
	}))
	defer ts.Close()

	c := NewClient()
	body, err := c.Download(ctx(t), ts.URL, WithDownloadRange(10, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "tail" {
		t.Fatalf("expected tail, got %q", string(data))
	}
	if gotRange != "bytes=10-" {
		t.Fatalf("expected bytes=10-, got %q", gotRange)
	}
}

func TestClient_Download_ClientLevelLimit(t *testing.T) {
	large := strings.Repeat("x", 1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(large))
	}))
	defer ts.Close()

	c := NewClient(WithDownloadBytesLimit(100))
	body, err := c.Download(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	_, err = io.ReadAll(body)
	if err == nil {
		t.Fatal("expected overflow error from client-level limit")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestPackageDownloadBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pkg download"))
	}))
	defer ts.Close()

	data, err := DownloadBytes(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "pkg download" {
		t.Fatalf("expected pkg download, got %q", string(data))
	}
}

func TestClient_Download_NilContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := NewClient()
	body, err := c.Download(nil, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "ok" {
		t.Fatalf("expected ok, got %q", string(data))
	}
}

func TestClient_DownloadFile_CreatesNestedDir(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("nested"))
	}))
	defer ts.Close()

	destPath := filepath.Join(t.TempDir(), "a", "b", "c", "deep.txt")
	c := NewClient()
	if err := c.DownloadFile(ctx(t), ts.URL, destPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "nested" {
		t.Fatalf("expected nested, got %q", string(data))
	}
}
