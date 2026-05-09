package netx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestEncodeMultipart_Error(t *testing.T) {
	fields := map[string][]string{"normal": {"data"}}
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

func TestBuildBody_URLEncodedDirect(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	resp, err := c.Do(nil, NewRequest(ts.URL, "POST",
		WithBody([]byte("foo=bar&baz=qux")),
		WithHeader("Content-Type", "application/x-www-form-urlencoded"),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
	}
	if gotBody != "foo=bar&baz=qux" {
		t.Errorf("expected original URL-encoded body, got %q", gotBody)
	}
}

func TestEncodeURLEncodedIfNeeded_AlreadyEncoded(t *testing.T) {
	reader, ct := EncodeURLEncodedIfNeeded([]byte("foo=bar&baz=qux"))
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content type, got %q", ct)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "foo=bar&baz=qux" {
		t.Errorf("expected original body, got %q", string(data))
	}
}

func TestEncodeURLEncodedIfNeeded_JSONToForm(t *testing.T) {
	reader, ct := EncodeURLEncodedIfNeeded([]byte(`{"foo":"bar"}`))
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content type, got %q", ct)
	}
	data, _ := io.ReadAll(reader)
	if !strings.Contains(string(data), "foo=bar") {
		t.Errorf("expected foo=bar, got %q", string(data))
	}
}

func TestEncodeURLEncodedIfNeeded_InvalidJSON(t *testing.T) {
	reader, ct := EncodeURLEncodedIfNeeded([]byte("just text"))
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content type, got %q", ct)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "just text" {
		t.Errorf("expected original text, got %q", string(data))
	}
}
