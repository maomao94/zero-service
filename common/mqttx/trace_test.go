package mqttx

import "testing"

func TestMessageCarrierHandlesNilMessage(t *testing.T) {
	carrier := NewMessageCarrier(nil)

	if got := carrier.Get("traceparent"); got != "" {
		t.Fatalf("expected empty header from nil message, got %s", got)
	}
	carrier.Set("traceparent", "value")
	if keys := carrier.Keys(); len(keys) != 0 {
		t.Fatalf("expected no keys from nil message, got %v", keys)
	}
}

func TestMessageCarrierHandlesNilHeaders(t *testing.T) {
	carrier := NewMessageCarrier(&Message{})

	if keys := carrier.Keys(); len(keys) != 0 {
		t.Fatalf("expected no keys from nil headers, got %v", keys)
	}
	carrier.Set("traceparent", "value")
	if got := carrier.Get("traceparent"); got != "value" {
		t.Fatalf("expected header value, got %s", got)
	}
}
