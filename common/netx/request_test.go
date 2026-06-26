package netx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWithJSONBody_MarshalError(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost,
		WithJSONBody(make(chan int)),
	)
	if req.OptionError == nil {
		t.Fatal("expected OptionError for unmarshalable type")
	}
	if !strings.Contains(req.OptionError.Error(), "marshal json body") {
		t.Errorf("unexpected error message: %v", req.OptionError)
	}
}

func TestWithBodyReader_DoesNotReadBeforeRequest(t *testing.T) {
	reader := &blockingReader{data: []byte("streamed")}
	req := NewRequest("http://example.com", http.MethodPost, WithBodyReader(reader))
	if req.OptionError != nil {
		t.Fatalf("unexpected option error: %v", req.OptionError)
	}
	if reader.reads != 0 {
		t.Fatalf("expected reader not read during option setup, got %d reads", reader.reads)
	}
}

func TestWithBodyReader_OptionDoesNotReadBeforeRequest(t *testing.T) {
	errReader := &errorReader{}
	req := NewRequest("http://example.com", http.MethodPost,
		WithBodyReader(errReader),
	)
	if req.OptionError != nil {
		t.Fatalf("unexpected option error: %v", req.OptionError)
	}
	if errReader.reads != 0 {
		t.Fatalf("expected reader not read during option setup, got %d reads", errReader.reads)
	}
}

func TestNewRequest_ValuesDontChangeAfterConstruction(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost,
		WithHeaders(http.Header{"X-Custom": {"original"}}),
		WithFormData(map[string][]string{"key": {"original"}}),
	)
	if h := req.Headers.Get("X-Custom"); h != "original" {
		t.Errorf("expected original header, got %q", h)
	}
	if v := req.FormData.Get("key"); v != "original" {
		t.Errorf("expected original form value, got %q", v)
	}
}

func TestRequest_BuilderChaining(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost).Header("X-KEY", "v").Query("q", "1")
	if req.Headers.Get("X-KEY") != "v" {
		t.Errorf("expected header from builder, got %q", req.Headers.Get("X-KEY"))
	}
	if req.QueryParams.Get("q") != "1" {
		t.Errorf("expected query param from builder, got %q", req.QueryParams.Get("q"))
	}
}

func TestClient_Do_RequestBuilderJSONFormRawReaderAndHeaderOverride(t *testing.T) {
	var gotJSONBody string
	var gotFormBody string
	var gotRawBody string
	var gotReaderBody string
	var gotContentType string
	var gotQuery string
	call := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch call {
		case 0:
			gotJSONBody = string(body)
		case 1:
			gotFormBody = string(body)
		case 2:
			gotRawBody = string(body)
			gotContentType = r.Header.Get("Content-Type")
			gotQuery = r.URL.RawQuery
		case 3:
			gotReaderBody = string(body)
		}
		call++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	if _, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodPost).JSON(map[string]string{"name": "zero"})); err != nil {
		t.Fatalf("json request failed: %v", err)
	}
	if _, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodPost).Form(url.Values{"a": {"1"}})); err != nil {
		t.Fatalf("form request failed: %v", err)
	}
	if _, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodPost).
		Header("Content-Type", "text/plain").
		Query("q", "go").
		Raw([]byte("raw body"))); err != nil {
		t.Fatalf("raw request failed: %v", err)
	}
	if _, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodPost).Reader(strings.NewReader("reader body"))); err != nil {
		t.Fatalf("reader request failed: %v", err)
	}

	if !strings.Contains(gotJSONBody, `"name":"zero"`) {
		t.Fatalf("expected json body, got %q", gotJSONBody)
	}
	if gotFormBody != "a=1" {
		t.Fatalf("expected form body, got %q", gotFormBody)
	}
	if gotRawBody != "raw body" {
		t.Fatalf("expected raw body, got %q", gotRawBody)
	}
	if gotContentType != "text/plain" {
		t.Fatalf("expected request content type override, got %q", gotContentType)
	}
	if gotQuery != "q=go" {
		t.Fatalf("expected query q=go, got %q", gotQuery)
	}
	if gotReaderBody != "reader body" {
		t.Fatalf("expected reader body, got %q", gotReaderBody)
	}
}

func TestWithBody_Clone(t *testing.T) {
	original := []byte("original")
	req := NewRequest("http://example.com", http.MethodPost, WithBody(original))
	original[0] = 'X'
	if req.Body[0] != 'o' {
		t.Fatalf("expected body to be cloned, got %q", string(req.Body))
	}
}

func TestWithBodyReader_NilDoesNothing(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost, WithBodyReader(nil))
	if req.BodyReader != nil {
		t.Fatal("expected nil BodyReader for nil input")
	}
	if req.bodyKind == bodyKindReader {
		t.Fatal("expected bodyKind not set for nil reader")
	}
}

func TestWithQueryParams_Clone(t *testing.T) {
	original := url.Values{"key": {"original"}}
	req := NewRequest("http://example.com", http.MethodGet, WithQueryParams(original))
	original.Set("key", "mutated")
	original.Add("extra", "val")
	if req.QueryParams.Get("key") != "original" {
		t.Fatalf("expected cloned query param, got %q", req.QueryParams.Get("key"))
	}
	if _, ok := req.QueryParams["extra"]; ok {
		t.Fatal("expected extra key not to leak into request")
	}
}

func TestRequest_BuilderHeadersMap(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodGet).HeadersMap(http.Header{"X-Bulk": {"v1", "v2"}})
	if len(req.Headers["X-Bulk"]) != 2 {
		t.Fatalf("expected 2 values, got %d", len(req.Headers["X-Bulk"]))
	}
	if req.Headers.Get("X-Bulk") != "v1" {
		t.Fatalf("expected v1, got %q", req.Headers.Get("X-Bulk"))
	}
}

func TestRequest_BuilderQueries(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodGet).Queries(url.Values{"a": {"1"}, "b": {"2"}})
	if req.QueryParams.Get("a") != "1" {
		t.Fatalf("expected a=1, got %q", req.QueryParams.Get("a"))
	}
	if req.QueryParams.Get("b") != "2" {
		t.Fatalf("expected b=2, got %q", req.QueryParams.Get("b"))
	}
}

func TestRequest_BuilderRaw(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost).Raw([]byte("raw"))
	if string(req.Body) != "raw" {
		t.Fatalf("expected raw body, got %q", string(req.Body))
	}
	if req.bodyKind != bodyKindRaw {
		t.Fatalf("expected bodyKindRaw, got %d", req.bodyKind)
	}
}

func TestRequest_BuilderForm(t *testing.T) {
	req := NewRequest("http://example.com", http.MethodPost).Form(url.Values{"k": {"v"}})
	if req.FormData.Get("k") != "v" {
		t.Fatalf("expected k=v, got %q", req.FormData.Get("k"))
	}
	if req.bodyKind != bodyKindForm {
		t.Fatalf("expected bodyKindForm, got %d", req.bodyKind)
	}
}

func TestRequest_BuilderReader(t *testing.T) {
	r := &blockingReader{data: []byte("data")}
	req := NewRequest("http://example.com", http.MethodPost).Reader(r)
	if req.BodyReader != r {
		t.Fatal("expected reader to be set")
	}
	if req.bodyKind != bodyKindReader {
		t.Fatalf("expected bodyKindReader, got %d", req.bodyKind)
	}
}
