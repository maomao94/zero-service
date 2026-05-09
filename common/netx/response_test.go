package netx

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
)

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
	resp := &Response{Err: ErrResponseTooLarge}
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

func TestDeCodeJSON_NonSuccess(t *testing.T) {
	resp := &Response{
		Data:       []byte(`{"code":1001}`),
		StatusCode: 400,
		Success:    false,
	}
	var target struct {
		Code int `json:"code"`
	}
	err := DecodeJSON(resp, &target)
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
