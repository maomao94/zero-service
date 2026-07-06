package gnetx

import "testing"

func TestRawSerializer(t *testing.T) {
	var s RawSerializer

	// Encode []byte
	out, err := s.Encode([]byte("hello"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(out) != "hello" {
		t.Fatalf("encode result = %q, want hello", out)
	}

	// Encode 非 []byte 应报错
	if _, err := s.Encode("string-not-bytes"); err == nil {
		t.Fatal("encode non-byte should error")
	}

	// Decode 返回 []byte 副本
	decoded, err := s.Decode([]byte("world"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b, ok := decoded.([]byte)
	if !ok || string(b) != "world" {
		t.Fatalf("decode result = %v, want []byte world", decoded)
	}
}

func TestJSONSerializer(t *testing.T) {
	var s JSONSerializer

	// Encode
	type ping struct {
		Type string `json:"type"`
		Val  int    `json:"val"`
	}
	out, err := s.Encode(ping{Type: "ping", Val: 42})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Decode
	decoded, err := s.Decode(out)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	m, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("decode result type = %T, want map[string]any", decoded)
	}
	if m["type"] != "ping" {
		t.Fatalf("type = %v, want ping", m["type"])
	}
	// JSON 数字默认解析为 float64
	if v, _ := m["val"].(float64); v != 42 {
		t.Fatalf("val = %v, want 42", m["val"])
	}
}
