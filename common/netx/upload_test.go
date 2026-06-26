package netx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_Upload(t *testing.T) {
	var gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("get form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "test.txt" {
			t.Errorf("expected filename test.txt, got %s", header.Filename)
		}
		data, _ := io.ReadAll(file)
		if string(data) != "hello world" {
			t.Errorf("expected file content, got %q", string(data))
		}
		if r.FormValue("desc") != "test upload" {
			t.Errorf("expected field desc, got %q", r.FormValue("desc"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file", FileName: "test.txt", Content: bytes.NewReader([]byte("hello world"))},
	}, map[string]string{"desc": "test upload"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
	}
	if !strings.Contains(gotCT, "multipart/form-data") {
		t.Errorf("expected multipart content type, got %q", gotCT)
	}
}

func TestClient_UploadBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		file, header, err := r.FormFile("attachment")
		if err != nil {
			t.Errorf("get form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "data.bin" {
			t.Errorf("expected filename data.bin, got %s", header.Filename)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.UploadBytes(ctx(t), ts.URL, "attachment", "data.bin", []byte("binary data"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_UploadFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	os.WriteFile(tmpFile, []byte("file content"), 0o644)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		file, _, err := r.FormFile("doc")
		if err != nil {
			t.Errorf("get form file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "file content" {
			t.Errorf("expected file content, got %q", string(data))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.UploadFile(ctx(t), ts.URL, tmpFile, "doc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Upload_PropagatesStreamingReadError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file", FileName: "bad.txt", Content: &errorAfterReader{}},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected direct error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failed response for streaming read error")
	}
	if !strings.Contains(resp.Err.Error(), "copy file content") {
		t.Fatalf("expected copy error, got %q", resp.Err.Error())
	}
}

func TestClient_Upload_UsesConfiguredUploadLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(WithUploadBytesLimit(3))
	resp, err := c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file", FileName: "large.txt", Content: strings.NewReader("abcd")},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected direct error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected upload limit failure")
	}
	if !strings.Contains(resp.Err.Error(), "upload body too large") {
		t.Fatalf("expected upload limit error, got %q", resp.Err.Error())
	}

	c = NewClient(WithUploadBytesLimit(0))
	resp, err = c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file", FileName: "large.txt", Content: strings.NewReader("abcd")},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected disabled limit direct error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected disabled limit success, got %q", resp.Err)
	}
}

func TestClient_Upload_MultipleFiles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		f1, h1, err := r.FormFile("file1")
		if err != nil {
			t.Errorf("get file1: %v", err)
		}
		defer f1.Close()
		if h1.Filename != "a.txt" {
			t.Errorf("expected a.txt, got %s", h1.Filename)
		}
		d1, _ := io.ReadAll(f1)
		if string(d1) != "content-a" {
			t.Errorf("expected content-a, got %q", string(d1))
		}
		f2, h2, err := r.FormFile("file2")
		if err != nil {
			t.Errorf("get file2: %v", err)
		}
		defer f2.Close()
		if h2.Filename != "b.txt" {
			t.Errorf("expected b.txt, got %s", h2.Filename)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file1", FileName: "a.txt", Content: bytes.NewReader([]byte("content-a"))},
		{FieldName: "file2", FileName: "b.txt", Content: bytes.NewReader([]byte("content-b"))},
	}, map[string]string{"meta": "multi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
	}
}

func TestClient_Upload_StreamError_StatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Upload(ctx(t), ts.URL, []FileUpload{
		{FieldName: "file", FileName: "bad.txt", Content: &errorAfterReader{}},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from broken pipe, got %d", resp.StatusCode)
	}
}

func TestClient_UploadFile_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	_, err := c.UploadFile(ctx(t), ts.URL, "/nonexistent/path/file.txt", "doc", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "open file") {
		t.Fatalf("expected open file error, got %v", err)
	}
}

func TestClient_Upload_WithRequestOptions(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Upload(ctx(t), ts.URL,
		[]FileUpload{{FieldName: "file", FileName: "test.txt", Content: bytes.NewReader([]byte("data"))}},
		nil,
		WithHeader("Authorization", "Bearer upload-token"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
	}
	if gotAuth != "Bearer upload-token" {
		t.Fatalf("expected auth header, got %q", gotAuth)
	}
}
