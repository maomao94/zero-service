package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/rest/httpc"
)

func TestNewClient_Default(t *testing.T) {
	c := NewClient()
	if c.engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if _, ok := c.engine.(*defaultEngine); !ok {
		t.Fatal("expected defaultEngine")
	}
	if c.timeout != DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultTimeout, c.timeout)
	}
}

func TestNewClient_WithHttpcService(t *testing.T) {
	svc := httpc.NewService("test")
	c := NewClient(WithHttpcService(svc))
	if _, ok := c.engine.(*httpcEngine); !ok {
		t.Fatal("expected httpcEngine")
	}
}

func TestNewClient_WithTimeout(t *testing.T) {
	c := NewClient(WithTimeout(5 * time.Second))
	if c.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", c.timeout)
	}
}

func TestNewClient_WithTLSConfig(t *testing.T) {
	cfg := &tls.Config{InsecureSkipVerify: true}
	c := NewClient(WithTLSConfig(cfg))
	if c.tlsConfig != cfg {
		t.Fatal("expected custom TLS config stored")
	}
}

func TestNewClient_WithHttpcAndTLS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secure":true}`))
	}))
	defer ts.Close()

	tlsCfg := &tls.Config{InsecureSkipVerify: true}
	tlsClient := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}
	svc := httpc.NewServiceWithClient("test-tls", tlsClient)
	c := NewClient(WithHttpcService(svc))
	if _, ok := c.engine.(*httpcEngine); !ok {
		t.Fatal("expected httpcEngine")
	}
	resp, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
}

func TestNewClient_WithDefaultHeaders(t *testing.T) {
	h := http.Header{"X-Custom": {"value"}}
	c := NewClient(WithDefaultHeaders(h))
	if c.headers.Get("X-Custom") != "value" {
		t.Fatal("expected default header X-Custom")
	}
}

func TestClient_Do_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, got error: %s", resp.Error)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestClient_Do_Post_JSON(t *testing.T) {
	var gotBody string
	var gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodPost, WithBody([]byte(`{"key":"value"}`))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
	if gotBody != `{"key":"value"}` {
		t.Errorf("expected body, got %q", gotBody)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("expected json content type, got %q", gotCT)
	}
}

func TestClient_Do_Post_FormData(t *testing.T) {
	var gotCT string
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodPost,
		WithFormData(url.Values{"key": {"value"}}),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
	if !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("expected form content type, got %q", gotCT)
	}
	if !strings.Contains(gotBody, "key=value") {
		t.Errorf("expected key=value in body, got %q", gotBody)
	}
}

func TestClient_Do_QueryParams(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodGet,
		WithQueryParams(url.Values{"page": {"1"}, "size": {"10"}}),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
	if !strings.Contains(gotQuery, "page=1") {
		t.Errorf("expected page=1 in query, got %q", gotQuery)
	}
}

func TestClient_Do_NilRequest(t *testing.T) {
	c := NewClient()
	_, err := c.Do(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestClient_Do_EmptyURL(t *testing.T) {
	c := NewClient()
	_, err := c.Do(context.Background(), &Request{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestClient_Do_WithHttpcService(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := httpc.NewService("test-svc")
	c := NewClient(WithHttpcService(svc))
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodPost, WithBody([]byte(`{"key":"value"}`))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
	if gotBody != `{"key":"value"}` {
		t.Errorf("expected body, got %q", gotBody)
	}
}

func TestClient_Do_WithTLS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secure":true}`))
	}))
	defer ts.Close()

	tlsCfg := &tls.Config{InsecureSkipVerify: true}
	c := NewClient(WithTLSConfig(tlsCfg))
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
}

func TestClient_Do_WithJSONBody(t *testing.T) {
	var gotBody string
	var gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	payload := map[string]string{"name": "test"}
	resp, err := c.Post(ctx(t), ts.URL, WithJSONBody(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("expected json content type, got %q", gotCT)
	}
	if !strings.Contains(gotBody, `"name":"test"`) {
		t.Errorf("expected json body, got %q", gotBody)
	}
}

func TestClient_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Post(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Put(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Put(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Delete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Delete(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Patch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Patch(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Head(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Head(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Options(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			t.Errorf("expected OPTIONS, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Options(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

// --- 包级函数测试 ---

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	resp, err := Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Post(ctx(t), ts.URL, WithBody([]byte(`{"key":"value"}`)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestSendRequest_PackageLevel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	resp, err := SendRequest(ctx(t), NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestSendRequest_WithHttpc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := httpc.NewService("test")
	resp, err := SendRequest(ctx(t), NewRequest(ts.URL, http.MethodGet), WithHttpcService(svc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestSendRequest_NilRequest(t *testing.T) {
	_, err := SendRequest(ctx(t), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestSendRequest_EmptyURL(t *testing.T) {
	_, err := SendRequest(ctx(t), &Request{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

// --- 文件上传测试 ---

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
		t.Errorf("expected success, error: %s", resp.Error)
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

// --- 文件下载测试 ---

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

// --- Response 工具测试 ---

func TestDecodeJSON_Success(t *testing.T) {
	resp := &Response{
		Data:    []byte(`{"name":"test","age":18}`),
		Success: true,
	}
	var target struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	err := DecodeJSON(resp, &target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "test" || target.Age != 18 {
		t.Errorf("unexpected result: %+v", target)
	}
}

func TestDecodeJSON_Error(t *testing.T) {
	resp := &Response{Error: "some error"}
	err := DecodeJSON(resp, &struct{}{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecodeJSON_Nil(t *testing.T) {
	err := DecodeJSON(nil, &struct{}{})
	if err == nil {
		t.Error("expected error for nil response")
	}
}

func TestFormatCostMs(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{100, "100ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{10000, "10.0s"},
	}
	for _, tt := range tests {
		got := FormatCostMs(tt.input)
		if got != tt.expected {
			t.Errorf("FormatCostMs(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- 编码工具测试 ---

func TestValidateAndFlatten(t *testing.T) {
	data, err := ValidateAndFlatten([]byte(`{"name":"test","age":18,"active":true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["name"][0] != "test" {
		t.Errorf("expected name=test, got %v", data["name"])
	}
	if data["age"][0] != "18" {
		t.Errorf("expected age=18, got %v", data["age"])
	}
}

func TestValidateAndFlatten_Nested(t *testing.T) {
	data, err := ValidateAndFlatten([]byte(`{"user":{"name":"admin","role":"manager"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["user.name"][0] != "admin" {
		t.Errorf("expected user.name=admin, got %v", data["user.name"])
	}
}

func TestValidateAndFlatten_InvalidJSON(t *testing.T) {
	_, err := ValidateAndFlatten([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid json")
	}
}

func TestValidateAndFlatten_Array(t *testing.T) {
	data, err := ValidateAndFlatten([]byte(`{"tags":["go","rust"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data["tags"]) != 2 {
		t.Errorf("expected 2 tags, got %v", data["tags"])
	}
}

func TestValidateAndFlatten_NullValue(t *testing.T) {
	data, err := ValidateAndFlatten([]byte(`{"key":null}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := data["key"]; ok {
		t.Error("expected null value to be skipped")
	}
}

func TestEncodeURLEncoded(t *testing.T) {
	encoded, err := EncodeURLEncoded([]byte(`{"foo":"bar","num":42}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(encoded, "foo=bar") {
		t.Errorf("expected foo=bar in %q", encoded)
	}
}

func TestEncodeURLEncoded_InvalidJSON(t *testing.T) {
	_, err := EncodeURLEncoded([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid json")
	}
}

func TestEncodeMultipart(t *testing.T) {
	fields := map[string][]string{"a": {"b"}, "c": {"d"}}
	reader, ct, err := EncodeMultipart(fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if !strings.Contains(ct, "multipart/form-data") {
		t.Errorf("expected multipart content type, got %q", ct)
	}
}

// --- WithBodyReader / buildBody / buildResponse 分支测试 ---

func TestClient_Do_WithBodyReader(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	reader := bytes.NewReader([]byte("streamed body"))
	resp, err := c.Post(ctx(t), ts.URL, WithBodyReader(reader))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
	if gotBody != "streamed body" {
		t.Errorf("expected streamed body, got %q", gotBody)
	}
}

func TestClient_Do_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(WithTimeout(100 * time.Millisecond))
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure for timeout")
	}
	if resp.Error == "" {
		t.Error("expected non-empty error message")
	}
	if resp.CostMs <= 0 {
		t.Error("expected positive cost")
	}
}

func TestClient_Do_LargeBody(t *testing.T) {
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = 'A'
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != len(largeData) {
		t.Errorf("expected %d bytes, got %d", len(largeData), len(resp.Data))
	}
}

func TestClient_Do_FormData_BuildBody(t *testing.T) {
	var gotCT string
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodPost,
		WithFormData(url.Values{"a": {"1"}, "b": {"2"}}),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Error)
	}
	if !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("expected form content type, got %q", gotCT)
	}
	if !strings.Contains(gotBody, "a=1") || !strings.Contains(gotBody, "b=2") {
		t.Errorf("expected form fields in body, got %q", gotBody)
	}
}

func TestClient_Do_BodyReader_NilBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Post(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

// --- 默认头合并测试 ---

func TestClient_DefaultHeaders_Merge(t *testing.T) {
	var gotAuth string
	var gotCustom string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCustom = r.Header.Get("X-Request-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(WithDefaultHeaders(http.Header{
		"Authorization": {"Bearer token123"},
	}))
	resp, err := c.Get(ctx(t), ts.URL, WithHeader("X-Request-Id", "req-001"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
	if gotAuth != "Bearer token123" {
		t.Errorf("expected auth header, got %q", gotAuth)
	}
	if gotCustom != "req-001" {
		t.Errorf("expected custom header, got %q", gotCustom)
	}
}

func TestClient_CostTracking(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CostMs < 0 {
		t.Errorf("expected non-negative cost, got %d", resp.CostMs)
	}
	if resp.CostFormatted == "" {
		t.Error("expected non-empty cost formatted")
	}
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}
