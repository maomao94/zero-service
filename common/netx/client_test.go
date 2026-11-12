package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
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
	if _, ok := c.engine.(*DefaultEngine); !ok {
		t.Fatal("expected DefaultEngine")
	}
	if c.maxResponseBytes != DefaultMaxResponseBytes {
		t.Fatalf("expected default max response bytes %d, got %d", DefaultMaxResponseBytes, c.maxResponseBytes)
	}
	if c.downloadBytesLimit != DefaultDownloadBytesLimit {
		t.Fatalf("expected default download bytes limit %d, got %d", DefaultDownloadBytesLimit, c.downloadBytesLimit)
	}
	if c.uploadBytesLimit != DefaultUploadBytesLimit {
		t.Fatalf("expected default upload bytes limit %d, got %d", DefaultUploadBytesLimit, c.uploadBytesLimit)
	}
}

func TestNewClient_WithByteLimits(t *testing.T) {
	c := NewClient(
		WithMaxResponseBytes(1),
		WithDownloadBytesLimit(2),
		WithUploadBytesLimit(3),
	)
	if c.maxResponseBytes != 1 {
		t.Fatalf("expected custom max response bytes, got %d", c.maxResponseBytes)
	}
	if c.downloadBytesLimit != 2 {
		t.Fatalf("expected custom download bytes limit, got %d", c.downloadBytesLimit)
	}
	if c.uploadBytesLimit != 3 {
		t.Fatalf("expected custom upload bytes limit, got %d", c.uploadBytesLimit)
	}
}

func TestNewClient_WithTLSConfig(t *testing.T) {
	cfg := &tls.Config{InsecureSkipVerify: true}
	c := NewClient(WithTLSConfig(cfg))
	if c.tlsConfig != cfg {
		t.Fatal("expected custom TLS config stored")
	}
}

func TestNewClient_WithDefaultHeaders(t *testing.T) {
	h := http.Header{"X-Custom": {"value"}}
	c := NewClient(WithDefaultHeaders(h))
	if c.headers.Get("X-Custom") != "value" {
		t.Fatal("expected default header X-Custom")
	}
}

func TestNewClient_WithCustomClientOption(t *testing.T) {
	c := NewClient(func(o *ClientOptions) {
		o.Headers = http.Header{"X-Option": {"value"}}
		o.MaxResponseBytes = 4
		o.DownloadBytesLimit = 5
		o.UploadBytesLimit = 6
	})

	if c.headers.Get("X-Option") != "value" {
		t.Fatal("expected custom option header")
	}
	if c.maxResponseBytes != 4 {
		t.Fatalf("expected custom max response bytes, got %d", c.maxResponseBytes)
	}
	if c.downloadBytesLimit != 5 {
		t.Fatalf("expected custom download bytes limit, got %d", c.downloadBytesLimit)
	}
	if c.uploadBytesLimit != 6 {
		t.Fatalf("expected custom upload bytes limit, got %d", c.uploadBytesLimit)
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
		t.Errorf("expected success, got error: %s", resp.Err)
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
		t.Errorf("expected success, error: %s", resp.Err)
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
		t.Errorf("expected success, error: %s", resp.Err)
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
		t.Errorf("expected success, error: %s", resp.Err)
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

	ctx, cancel := context.WithTimeout(ctx(t), 100*time.Millisecond)
	defer cancel()
	c := NewClient()
	resp, err := c.Get(ctx, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure for timeout")
	}
	if resp.Err == nil {
		t.Error("expected non-empty error message")
	}
	if resp.CostMs <= 0 {
		t.Error("expected positive cost")
	}
}

func TestClient_Do_LargeBody(t *testing.T) {
	largeData := make([]byte, 10*1024*1024)
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

func TestClient_Do_WithEngine(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := httpc.NewService("test-svc")
	c := NewClient(WithEngine(NewHTTPEngine(svc)))
	resp, err := c.Do(context.Background(), NewRequest(ts.URL, http.MethodPost, WithBody([]byte(`{"key":"value"}`))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
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
		t.Errorf("expected success, error: %s", resp.Err)
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

func TestClient_Do_UsesContextTimeoutWithoutHTTPClientTimeout(t *testing.T) {
	c := NewClient()
	eng, ok := c.engine.(*DefaultEngine)
	if !ok {
		t.Fatal("expected default engine")
	}
	if eng.client.Timeout != 0 {
		t.Fatalf("expected http.Client timeout disabled, got %v", eng.client.Timeout)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(300 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	reqCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	resp, err := c.Get(reqCtx, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected timeout response")
	}
	if resp.Err == nil {
		t.Fatal("expected error for timeout")
	}
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 gateway timeout, got %d", resp.StatusCode)
	}
}

func TestClient_Do_RespectsExistingContextDeadline(t *testing.T) {
	c := NewClient()
	reqCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	sawDone := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			close(sawDone)
		case <-time.After(time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	resp, err := c.Get(reqCtx, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Err == nil {
		t.Fatal("expected error for timeout")
	}
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 gateway timeout, got %d", resp.StatusCode)
	}
	select {
	case <-sawDone:
	case <-time.After(time.Second):
		t.Fatal("expected caller deadline to cancel request")
	}
}

func TestClient_Do_UsesContextWithoutDeadline(t *testing.T) {
	c := NewClient()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); ok {
			t.Error("expected no derived deadline")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %s", resp.Err)
	}
}

func TestClient_Do_OptionError(t *testing.T) {
	c := NewClient()
	req := NewRequest("http://example.com", http.MethodPost,
		WithJSONBody(make(chan int)),
	)
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from option error")
	}
}

func TestClient_Do_UsesConfiguredResponseLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("abcd"))
	}))
	defer ts.Close()

	c := NewClient(WithMaxResponseBytes(3))
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected response limit failure")
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 for oversized body, got %d", resp.StatusCode)
	}
	if resp.Err == nil || !strings.Contains(resp.Err.Error(), "response body too large") {
		t.Fatalf("expected response limit error, got %q", resp.Err)
	}

	c = NewClient(WithMaxResponseBytes(0))
	resp, err = c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected disabled limit error: %v", err)
	}
	if string(resp.Data) != "abcd" {
		t.Fatalf("expected full response body, got %q", string(resp.Data))
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

func TestWithDefaultHeaders_Clone(t *testing.T) {
	original := http.Header{"Authorization": {"Bearer original"}}
	c := NewClient(WithDefaultHeaders(original))

	original.Set("X-Evil", "injected")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Evil") != "" {
			t.Error("X-Evil should not be present")
		}
		if r.Header.Get("Authorization") != "Bearer original" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
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
		t.Errorf("expected success, error: %s", resp.Err)
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

func TestPackageGet(t *testing.T) {
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

func TestPackagePost(t *testing.T) {
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
	resp, err := SendRequest(ctx(t), NewRequest(ts.URL, http.MethodGet), WithEngine(NewHTTPEngine(svc)))
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

func TestClient_Do_NilContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := NewClient()
	//lint:ignore SA1012 故意传 nil，验证 Client 的 nil context 回退行为
	resp, err := c.Do(nil, NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success with nil context fallback")
	}
}

func TestClient_Do_ResponseHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Resp", "value123")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Headers == nil {
		t.Fatal("expected non-nil response headers")
	}
	if resp.Headers.Get("X-Custom-Resp") != "value123" {
		t.Fatalf("expected X-Custom-Resp=value123, got %q", resp.Headers.Get("X-Custom-Resp"))
	}
}

func TestClient_Do_RequestHeaderOverridesClientHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer override" {
			t.Errorf("expected override, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(WithDefaultHeaders(http.Header{
		"Authorization": {"Bearer default"},
	}))
	resp, err := c.Get(ctx(t), ts.URL, WithHeader("Authorization", "Bearer override"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
}

func TestClient_Do_NetworkError_503(t *testing.T) {
	c := NewClient()
	resp, err := c.Get(ctx(t), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure for unreachable host")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 service unavailable, got %d", resp.StatusCode)
	}
}

func TestClient_Do_ContextCanceled_400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	c := NewClient()
	resp, err := c.Get(reqCtx, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure for canceled context")
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 bad request for cancel, got %d", resp.StatusCode)
	}
}

func TestPackagePut(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Put(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestPackageDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Delete(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestPackagePatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Patch(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestPackageHead(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Head(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestPackageOptions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			t.Errorf("expected OPTIONS, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := Options(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestClient_Do_InvalidURL(t *testing.T) {
	c := NewClient()
	_, err := c.Do(ctx(t), NewRequest("://invalid-url", http.MethodGet))
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewClient_WithHTTPClientOption(t *testing.T) {
	var gotRedirect int
	c := NewClient(
		WithHTTPClientOption(func(hc *http.Client) {
			hc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				gotRedirect++
				return http.ErrUseLastResponse
			}
		}),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := c.Get(ctx(t), ts.URL+"/redirect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRedirect == 0 {
		t.Fatal("expected CheckRedirect to be called")
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 (redirect not followed), got %d", resp.StatusCode)
	}
}

func TestNewClient_WithHTTPClientOption_Jar(t *testing.T) {
	c := NewClient(
		WithHTTPClientOption(func(hc *http.Client) {
			hc.Jar, _ = cookiejar.New(nil)
		}),
	)
	eng, ok := c.engine.(*DefaultEngine)
	if !ok {
		t.Fatal("expected DefaultEngine")
	}
	if eng.client.Jar == nil {
		t.Fatal("expected cookie jar to be set")
	}
}

type echoResult struct {
	Method      string
	ContentType string
	Body        string
	RawQuery    string
	Header      http.Header
}

func newEchoServer(t *testing.T, got *echoResult) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = echoResult{
			Method:      r.Method,
			ContentType: r.Header.Get("Content-Type"),
			Body:        string(b),
			RawQuery:    r.URL.RawQuery,
			Header:      r.Header.Clone(),
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestClient_Do_BodyOptionLastWins(t *testing.T) {
	tests := []struct {
		name     string
		opts     []RequestOption
		wantCT   string
		wantBody string
	}{
		{
			name:     "form then json",
			opts:     []RequestOption{WithFormData(url.Values{"a": {"1"}}), WithJSONBody(map[string]int{"b": 2})},
			wantCT:   "application/json",
			wantBody: `{"b":2}`,
		},
		{
			name:     "json then form",
			opts:     []RequestOption{WithJSONBody(map[string]int{"b": 2}), WithFormData(url.Values{"a": {"1"}})},
			wantCT:   "application/x-www-form-urlencoded",
			wantBody: "a=1",
		},
		{
			name:     "form then raw",
			opts:     []RequestOption{WithFormData(url.Values{"a": {"1"}}), WithBody([]byte(`{"b":2}`))},
			wantCT:   "application/json",
			wantBody: `{"b":2}`,
		},
		{
			name:     "form then reader",
			opts:     []RequestOption{WithFormData(url.Values{"a": {"1"}}), WithBodyReader(strings.NewReader("stream"))},
			wantBody: "stream",
		},
		{
			name:     "reader then json",
			opts:     []RequestOption{WithBodyReader(strings.NewReader("stream")), WithJSONBody(map[string]int{"b": 2})},
			wantCT:   "application/json",
			wantBody: `{"b":2}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got echoResult
			ts := newEchoServer(t, &got)
			resp, err := NewClient().Post(ctx(t), ts.URL, tt.opts...)
			if err != nil || resp.Err != nil {
				t.Fatalf("unexpected error: %v %v", err, resp.Err)
			}
			if got.Body != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, got.Body)
			}
			if got.ContentType != tt.wantCT {
				t.Fatalf("expected content type %q, got %q", tt.wantCT, got.ContentType)
			}
		})
	}
}

func TestClient_Do_EmptyMethodDefaultsToGet(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	if _, err := NewClient().Do(ctx(t), &Request{URL: ts.URL}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != http.MethodGet {
		t.Fatalf("expected GET, got %s", got.Method)
	}
}

func TestClient_Do_EmptyFormData(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	if _, err := NewClient().Post(ctx(t), ts.URL, WithFormData(nil)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body != "" {
		t.Fatalf("expected empty body, got %q", got.Body)
	}
	if got.ContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form content type, got %q", got.ContentType)
	}
}

func TestClient_Do_DirectFieldAssignment(t *testing.T) {
	t.Run("form data", func(t *testing.T) {
		var got echoResult
		ts := newEchoServer(t, &got)
		req := &Request{URL: ts.URL, Method: http.MethodPost, FormData: url.Values{"k": {"v"}}}
		if _, err := NewClient().Do(ctx(t), req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Body != "k=v" || got.ContentType != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected request: %+v", got)
		}
	})
	t.Run("body reader", func(t *testing.T) {
		var got echoResult
		ts := newEchoServer(t, &got)
		req := &Request{URL: ts.URL, Method: http.MethodPost, BodyReader: strings.NewReader("data"), ContentType: "text/plain"}
		if _, err := NewClient().Do(ctx(t), req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Body != "data" || got.ContentType != "text/plain" {
			t.Fatalf("unexpected request: %+v", got)
		}
	})
	t.Run("raw body", func(t *testing.T) {
		var got echoResult
		ts := newEchoServer(t, &got)
		req := &Request{URL: ts.URL, Method: http.MethodPost, Body: []byte(`{"a":1}`)}
		if _, err := NewClient().Do(ctx(t), req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Body != `{"a":1}` || got.ContentType != "application/json" {
			t.Fatalf("unexpected request: %+v", got)
		}
	})
}

func TestClient_Do_RawBodyWithFormContentTypeEncodesJSON(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	resp, err := NewClient().Post(ctx(t), ts.URL,
		WithHeader("Content-Type", "application/x-www-form-urlencoded"),
		WithBody([]byte(`{"a":"1","n":{"b":2}}`)),
	)
	if err != nil || resp.Err != nil {
		t.Fatalf("unexpected error: %v %v", err, resp.Err)
	}
	values, err := url.ParseQuery(got.Body)
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if values.Get("a") != "1" || values.Get("n.b") != "2" {
		t.Fatalf("expected flattened form body, got %q", got.Body)
	}
}

func TestClient_Do_EmptyRawBodySendsNoContentType(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	if _, err := NewClient().Post(ctx(t), ts.URL, WithBody(nil)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body != "" || got.ContentType != "" {
		t.Fatalf("expected no body and no content type, got %+v", got)
	}
}

func TestClient_Do_MultipartContentTypeOverridesHeader(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	resp, err := NewClient().UploadBytes(ctx(t), ts.URL, "file", "a.txt", []byte("x"), nil,
		WithHeader("Content-Type", "application/json"),
	)
	if err != nil || resp.Err != nil {
		t.Fatalf("unexpected error: %v %v", err, resp.Err)
	}
	if !strings.HasPrefix(got.ContentType, "multipart/form-data; boundary=") {
		t.Fatalf("expected multipart content type, got %q", got.ContentType)
	}
}

func TestClient_Do_QueryParamsMergeWithURL(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	if _, err := NewClient().Get(ctx(t), ts.URL+"?x=1&y=2", WithQueryParams(url.Values{"x": {"3"}})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q, _ := url.ParseQuery(got.RawQuery)
	if strings.Join(q["x"], ",") != "1,3" || q.Get("y") != "2" {
		t.Fatalf("expected merged query, got %q", got.RawQuery)
	}
}

func TestClient_Do_QueryBuilderAppends(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	req := NewRequest(ts.URL, http.MethodGet).Query("k", "1").Query("k", "2")
	if _, err := NewClient().Do(ctx(t), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q, _ := url.ParseQuery(got.RawQuery)
	if strings.Join(q["k"], ",") != "1,2" {
		t.Fatalf("expected repeated query values, got %q", got.RawQuery)
	}
}

func TestClient_Do_DefaultHeadersMultiValue(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	c := NewClient(WithDefaultHeaders(http.Header{
		"X-Multi": {"a", "b"},
		"X-Keep":  {"client"},
	}))
	if _, err := c.Get(ctx(t), ts.URL, WithHeaders(http.Header{"X-Multi": {"c"}})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(got.Header.Values("X-Multi"), ",") != "c" {
		t.Fatalf("expected request header to replace all default values, got %v", got.Header.Values("X-Multi"))
	}
	if strings.Join(got.Header.Values("X-Keep"), ",") != "client" {
		t.Fatalf("expected untouched default header, got %v", got.Header.Values("X-Keep"))
	}
}

func TestClient_Do_DefaultHeadersNotMutatedByRequest(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	c := NewClient(WithDefaultHeaders(http.Header{"X-Token": {"default"}}))
	if _, err := c.Get(ctx(t), ts.URL, WithHeader("X-Token", "override")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := c.Get(ctx(t), ts.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Header.Get("X-Token") != "default" {
		t.Fatalf("expected default header restored on next request, got %q", got.Header.Get("X-Token"))
	}
}

func TestClient_Do_Non2xxKeepsBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reason", "missing")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer ts.Close()

	resp, err := NewClient().Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success || resp.Err != nil {
		t.Fatalf("expected non-success without transport error, got %+v", resp)
	}
	if resp.StatusCode != http.StatusNotFound || string(resp.Data) != "not found" {
		t.Fatalf("unexpected response: %d %q", resp.StatusCode, resp.Data)
	}
	if resp.Headers.Get("X-Reason") != "missing" {
		t.Fatal("expected response headers on non-2xx")
	}
}

func TestClient_Do_ResponseTooLargeIsTyped(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("abcdef"))
	}))
	defer ts.Close()

	resp, err := NewClient(WithMaxResponseBytes(2)).Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(resp.Err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", resp.Err)
	}
	if resp.Data != nil || resp.Headers != nil {
		t.Fatal("expected no partial data on oversized response")
	}
}

func TestClient_Do_ResponseExactlyAtLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("abc"))
	}))
	defer ts.Close()

	resp, err := NewClient(WithMaxResponseBytes(3)).Get(ctx(t), ts.URL)
	if err != nil || resp.Err != nil {
		t.Fatalf("unexpected error: %v %v", err, resp.Err)
	}
	if string(resp.Data) != "abc" {
		t.Fatalf("expected full body, got %q", resp.Data)
	}
}

func TestClient_Do_ResponseBodyReadError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}))
	defer ts.Close()

	resp, err := NewClient().Get(ctx(t), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure on truncated body")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for read error, got %d", resp.StatusCode)
	}
	if resp.Err == nil || !strings.Contains(resp.Err.Error(), "read response body") {
		t.Fatalf("expected read response body error, got %v", resp.Err)
	}
	if errors.Is(resp.Err, ErrResponseTooLarge) {
		t.Fatal("read error must not be reported as too large")
	}
}

type engineFunc func(*http.Request) (*http.Response, error)

func (f engineFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestClient_Do_EngineError(t *testing.T) {
	engineErr := errors.New("breaker open")
	c := NewClient(WithEngine(engineFunc(func(*http.Request) (*http.Response, error) {
		return nil, engineErr
	})))
	resp, err := c.Get(ctx(t), "http://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(resp.Err, engineErr) {
		t.Fatalf("expected engine error, got %v", resp.Err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || resp.Success {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClient_Do_EngineReceivesBuiltRequest(t *testing.T) {
	var captured *http.Request
	var capturedBody string
	c := NewClient(
		WithDefaultHeaders(http.Header{"X-Default": {"d"}}),
		WithEngine(engineFunc(func(req *http.Request) (*http.Response, error) {
			captured = req
			b, _ := io.ReadAll(req.Body)
			capturedBody = string(b)
			return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
		})),
	)
	reqCtx := context.WithValue(ctx(t), struct{}{}, "v")
	resp, err := c.Put(reqCtx, "http://example.com/path", WithJSONBody(map[string]string{"k": "v"}), WithHeader("X-Req", "r"))
	if err != nil || resp.Err != nil {
		t.Fatalf("unexpected error: %v %v", err, resp.Err)
	}
	if captured.Method != http.MethodPut || captured.URL.Path != "/path" {
		t.Fatalf("unexpected request: %s %s", captured.Method, captured.URL)
	}
	if captured.Header.Get("X-Default") != "d" || captured.Header.Get("X-Req") != "r" {
		t.Fatalf("unexpected headers: %v", captured.Header)
	}
	if captured.Context().Value(struct{}{}) != "v" {
		t.Fatal("expected request context to be propagated")
	}
	if capturedBody != `{"k":"v"}` {
		t.Fatalf("unexpected body: %q", capturedBody)
	}
	if !resp.Success || resp.StatusCode != http.StatusCreated || string(resp.Data) != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClient_Do_ResponseHeadersCloned(t *testing.T) {
	upstream := http.Header{"X-A": {"1"}}
	c := NewClient(WithEngine(engineFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: upstream, Body: io.NopCloser(strings.NewReader(""))}, nil
	})))
	resp, err := c.Get(ctx(t), "http://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Headers.Set("X-A", "changed")
	if upstream.Get("X-A") != "1" {
		t.Fatal("expected response headers to be cloned")
	}
}

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestClient_Do_ClosesResponseBody(t *testing.T) {
	for _, limit := range []int64{0, 2} {
		body := &closeTrackingBody{Reader: strings.NewReader("abcdef")}
		c := NewClient(
			WithMaxResponseBytes(limit),
			WithEngine(engineFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body}, nil
			})),
		)
		if _, err := c.Get(ctx(t), "http://example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !body.closed {
			t.Fatalf("expected response body closed (limit=%d)", limit)
		}
	}
}

func TestSendRequest_WithClientOptions(t *testing.T) {
	var got echoResult
	ts := newEchoServer(t, &got)
	resp, err := SendRequest(ctx(t), NewRequest(ts.URL, http.MethodGet),
		WithDefaultHeaders(http.Header{"X-Send": {"yes"}}),
	)
	if err != nil || !resp.Success {
		t.Fatalf("unexpected result: %v %+v", err, resp)
	}
	if got.Header.Get("X-Send") != "yes" {
		t.Fatal("expected client option applied")
	}
}

func TestClient_Do_ConcurrentRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Query().Get("i")))
	}))
	defer ts.Close()

	c := NewClient(WithDefaultHeaders(http.Header{"X-Shared": {"v"}}))
	const n = 20
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			want := strconv.Itoa(i)
			resp, err := c.Get(ctx(t), ts.URL, WithQueryParams(url.Values{"i": {want}}), WithHeader("X-Req", want))
			if err == nil && string(resp.Data) != want {
				err = fmt.Errorf("expected %s, got %q", want, resp.Data)
			}
			errCh <- err
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
}
