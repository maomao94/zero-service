package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildBody(t *testing.T) {
	body, err := buildBody("pod.list", `{"limit":10,"offset":0}`, "req-001", 1761120000000, "nonce-001")
	if err != nil {
		t.Fatalf("build body: %v", err)
	}

	want := `{"method":"pod.list","data":{"limit":10,"offset":0},"tid":"req-001","timestamp":1761120000000,"nonce":"nonce-001"}`
	if string(body) != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
	if !json.Valid(body) {
		t.Fatal("body should be valid JSON")
	}
}

func TestBuildBodyOmitsOptionalFields(t *testing.T) {
	body, err := buildBody("pod.list", "", "", 1761120000000, "nonce-001")
	if err != nil {
		t.Fatalf("build body: %v", err)
	}

	want := `{"method":"pod.list","timestamp":1761120000000,"nonce":"nonce-001"}`
	if string(body) != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestBuildCurl(t *testing.T) {
	curl := buildCurl("http://127.0.0.1:11005/v1/gateway", "app_test", "sec_test", "abc123", `{"method":"pod.list"}`)
	for _, want := range []string{
		"http://127.0.0.1:11005/v1/gateway",
		"Authorization: Basic ",
		"X-Sign: v1:abc123",
		`--data-binary '{"method":"pod.list"}'`,
	} {
		if !strings.Contains(curl, want) {
			t.Fatalf("curl missing %q: %s", want, curl)
		}
	}
}

func TestShellQuote(t *testing.T) {
	if got, want := shellQuote("a'b"), `'a'\''b'`; got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := map[string]string{
		"":                      defaultGatewayURL,
		"127.0.0.1:11005":       "http://127.0.0.1:11005",
		"http://host:11005/":    "http://host:11005",
		"https://host/openapi/": "https://host/openapi",
	}
	for input, want := range tests {
		if got := normalizeBaseURL(input); got != want {
			t.Fatalf("normalizeBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}
