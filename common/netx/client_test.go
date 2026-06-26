package netx

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
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
