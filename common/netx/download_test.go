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
	if err == nil || !strings.Contains(err.Error(), "download body too large") {
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
	if err == nil || !strings.Contains(err.Error(), "download body too large") {
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
