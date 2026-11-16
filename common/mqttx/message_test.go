package mqttx

import "testing"

func TestNewMessageInitializesHeaders(t *testing.T) {
	msg := NewMessage("device/1", []byte("payload"))

	if msg.Topic != "device/1" || string(msg.Payload) != "payload" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Headers == nil {
		t.Fatal("expected headers to be initialized")
	}
	if got := msg.GetHeader("traceparent"); got != "" {
		t.Fatalf("GetHeader = %q, want empty", got)
	}

	msg.SetHeader("traceparent", "00-trace-span-01")
	if got := msg.GetHeader("traceparent"); got != "00-trace-span-01" {
		t.Fatalf("GetHeader = %q, want traceparent", got)
	}
}

func TestMessageHeaderAccessorsHandleNilMap(t *testing.T) {
	msg := &Message{}

	if got := msg.GetHeader("any"); got != "" {
		t.Fatalf("GetHeader on nil map = %q, want empty", got)
	}

	msg.SetHeader("any", "value")
	if got := msg.GetHeader("any"); got != "value" {
		t.Fatalf("GetHeader = %q, want value", got)
	}
}
