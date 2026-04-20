package invoke

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"zero-service/app/trigger/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpc"
)

func TestHTTPInvoker_URLEncoded(t *testing.T) {
	var gotCT string
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "form-1",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:       []byte(`{"name":"test","age":18}`),
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("expected Content-Type to contain application/x-www-form-urlencoded, got %q", gotCT)
	}
	if !strings.Contains(gotBody, "name=test") {
		t.Errorf("expected body to contain 'name=test', got %q", gotBody)
	}
	if !strings.Contains(gotBody, "age=18") {
		t.Errorf("expected body to contain 'age=18', got %q", gotBody)
	}
}

func TestHTTPInvoker_Multipart(t *testing.T) {
	var gotCT string
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "form-2",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Headers:    map[string]string{"Content-Type": "multipart/form-data"},
		Body:       []byte(`{"username":"admin","password":"123456"}`),
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(gotCT, "multipart/form-data") {
		t.Errorf("expected Content-Type to contain multipart/form-data, got %q", gotCT)
	}
	if !strings.Contains(gotBody, "username") || !strings.Contains(gotBody, "admin") {
		t.Errorf("expected body to contain username=admin, got %q", gotBody)
	}
	if !strings.Contains(gotBody, "password") || !strings.Contains(gotBody, "123456") {
		t.Errorf("expected body to contain password=123456, got %q", gotBody)
	}
}

func TestHTTPInvoker_JSONBody(t *testing.T) {
	var gotCT string
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "json-1",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Body:       []byte(`{"key":"value"}`),
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %q", gotCT)
	}
	if gotBody != `{"key":"value"}` {
		t.Errorf("expected body to be original JSON, got %q", gotBody)
	}
}

func TestHTTPInvoker_FormWithInvalidJSON(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "invalid-1",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:       []byte(`not-json`),
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if gotBody != "not-json" {
		t.Errorf("expected body to fallback to raw bytes, got %q", gotBody)
	}
}

func TestHTTPInvoker_NoBodyWithFormContentType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "no-body-1",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestRun_FormURLSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "wrong content type: %s", r.Header.Get("Content-Type"))
			return
		}
		r.ParseForm()
		if r.FormValue("key") != "value" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing form field"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := newTestSvcCtx()
	tasks := []*Task{
		{
			ID:         "form-task",
			Protocol:   "http",
			HTTPMethod: "POST",
			URL:        ts.URL,
			Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			Body:       []byte(`{"key":"value"}`),
		},
	}

	results := Run(context.Background(), sc, tasks, 0, false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success, got error: %s, data: %s", results[0].Error, string(results[0].Data))
	}
}

func TestHTTPInvoker_NestedFormURL(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotBody = r.Form.Encode()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := &svc.ServiceContext{Httpc: httpc.NewService("form-test")}
	h := &HTTPInvoker{}
	result := h.Execute(context.Background(), sc, &Task{
		ID:         "nested-form",
		Protocol:   "http",
		HTTPMethod: "POST",
		URL:        ts.URL,
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:       []byte(`{"user":{"name":"admin","age":25},"tags":["go","rust"]}`),
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	parsed := parseURLEncoded(gotBody)
	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if parsed["user.name"][0] != "admin" {
		t.Errorf("expected user.name=admin, got %v", parsed["user.name"])
	}
	if parsed["user.age"][0] != "25" {
		t.Errorf("expected user.age=25, got %v", parsed["user.age"])
	}
	if len(parsed["tags"]) != 2 || parsed["tags"][0] != "go" || parsed["tags"][1] != "rust" {
		t.Errorf("expected tags=[go,rust], got %v", parsed["tags"])
	}
}

func TestHTTPInvoker_ValidateAndFlatten_Delegate(t *testing.T) {
	data, err := ValidateAndFlatten([]byte(`{"name":"test","nested":{"key":"val"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["name"][0] != "test" {
		t.Errorf("expected name=test, got %v", data["name"])
	}
	if data["nested.key"][0] != "val" {
		t.Errorf("expected nested.key=val, got %v", data["nested.key"])
	}
}

func TestHTTPInvoker_EncodeURLEncoded_Delegate(t *testing.T) {
	encoded, err := EncodeURLEncoded([]byte(`{"foo":"bar"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(encoded, "foo=bar") {
		t.Errorf("expected foo=bar in %q", encoded)
	}
}

func parseURLEncoded(s string) map[string][]string {
	result := make(map[string][]string)
	parts := strings.Split(s, "&")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = append(result[kv[0]], kv[1])
		}
	}
	return result
}
