package netx

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestResponseJSON_Success(t *testing.T) {
	resp := &Response{
		Data:    []byte(`{"name":"test","age":18}`),
		Success: true,
	}
	var target struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	err := resp.JSON(&target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "test" || target.Age != 18 {
		t.Errorf("unexpected result: %+v", target)
	}
}

func TestResponseJSON_Error(t *testing.T) {
	resp := &Response{Err: ErrResponseTooLarge}
	err := resp.JSON(&struct{}{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestResponseJSON_Nil(t *testing.T) {
	var resp *Response
	err := resp.JSON(&struct{}{})
	if err == nil {
		t.Error("expected error for nil response")
	}
}

func TestResponseJSON_NonSuccess(t *testing.T) {
	resp := &Response{
		Data:       []byte(`{"code":1001}`),
		StatusCode: 400,
		Success:    false,
	}
	var target struct {
		Code int `json:"code"`
	}
	err := resp.JSON(&target)
	if err == nil {
		t.Error("expected error for non-success response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention status code, got: %v", err)
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

func TestFormatCostMs_EdgeCases(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0ms"},
		{-1, "-1ms"},
		{-1500, "-1500ms"},
	}
	for _, tt := range tests {
		got := FormatCostMs(tt.input)
		if got != tt.expected {
			t.Errorf("FormatCostMs(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResponseDecodeHelpers(t *testing.T) {
	type sample struct {
		XMLName xml.Name `xml:"sample" json:"-"`
		Name    string   `xml:"name" json:"name"`
	}

	jsonResp := &Response{StatusCode: 200, Success: true, Headers: http.Header{"Content-Type": {"application/json"}}, Data: []byte(`{"name":"zero"}`)}
	var fromJSON sample
	if err := jsonResp.JSON(&fromJSON); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	if fromJSON.Name != "zero" {
		t.Fatalf("expected json name zero, got %q", fromJSON.Name)
	}

	xmlResp := &Response{StatusCode: 200, Success: true, Headers: http.Header{"Content-Type": {"application/xml"}}, Data: []byte(`<sample><name>zero</name></sample>`)}
	var fromXML sample
	if err := xmlResp.XML(&fromXML); err != nil {
		t.Fatalf("xml decode failed: %v", err)
	}
	if fromXML.Name != "zero" {
		t.Fatalf("expected xml name zero, got %q", fromXML.Name)
	}
	text, err := (&Response{StatusCode: 200, Success: true, Data: []byte("plain")}).Text()
	if err != nil {
		t.Fatalf("text failed: %v", err)
	}
	if text != "plain" {
		t.Fatalf("expected plain text, got %q", text)
	}

	var autoJSON sample
	if err := (&Response{StatusCode: 200, Success: true, Headers: http.Header{"Content-Type": {"text/plain"}}, Data: []byte(` {"name":"sniff"}`)}).Decode(&autoJSON); err != nil {
		t.Fatalf("auto json decode failed: %v", err)
	}
	if autoJSON.Name != "sniff" {
		t.Fatalf("expected sniff name, got %q", autoJSON.Name)
	}

	var unsupported sample
	if err := (&Response{StatusCode: 200, Success: true, Headers: http.Header{"Content-Type": {"application/octet-stream"}}, Data: []byte("raw")}).Decode(&unsupported); err == nil {
		t.Fatal("expected unsupported decode error")
	}
}

func TestClassifyNetErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"deadline exceeded", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"canceled", context.Canceled, http.StatusBadRequest},
		{"generic network error", errors.New("connection refused"), http.StatusServiceUnavailable},
		{"wrapped deadline", fmt.Errorf("ctx: %w", context.DeadlineExceeded), http.StatusGatewayTimeout},
		{"wrapped canceled", fmt.Errorf("ctx: %w", context.Canceled), http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyNetErr(tt.err)
			if got != tt.want {
				t.Fatalf("classifyNetErr(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestResponse_XML_Error(t *testing.T) {
	resp := &Response{Err: ErrResponseTooLarge}
	err := resp.XML(&struct{}{})
	if err == nil {
		t.Fatal("expected error for response with Err")
	}
}

func TestResponse_Text_Error(t *testing.T) {
	resp := &Response{Err: ErrResponseTooLarge}
	_, err := resp.Text()
	if err == nil {
		t.Fatal("expected error for response with Err")
	}
}

func TestResponseDecode_TextString(t *testing.T) {
	resp := &Response{
		StatusCode: 200,
		Success:    true,
		Headers:    http.Header{"Content-Type": {"text/plain"}},
		Data:       []byte("hello text"),
	}
	var s string
	if err := resp.Decode(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "hello text" {
		t.Fatalf("expected 'hello text', got %q", s)
	}
}

func TestResponseDecode_TextNonStringTarget(t *testing.T) {
	resp := &Response{
		StatusCode: 200,
		Success:    true,
		Headers:    http.Header{"Content-Type": {"text/plain"}},
		Data:       []byte("hello text"),
	}
	var n int
	if err := resp.Decode(&n); err == nil {
		t.Fatal("expected error for text decode with non-string target")
	}
}

func TestResponseDecode_XMLByContent(t *testing.T) {
	resp := &Response{
		StatusCode: 200,
		Success:    true,
		Data:       []byte(`<sample><name>xml</name></sample>`),
	}
	type sample struct {
		XMLName xml.Name `xml:"sample"`
		Name    string   `xml:"name"`
	}
	var s sample
	if err := resp.Decode(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "xml" {
		t.Fatalf("expected xml, got %q", s.Name)
	}
}
