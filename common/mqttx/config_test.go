package mqttx

import "testing"

func TestAdjustConfigDefaults(t *testing.T) {
	cfg := MqttConfig{}

	adjustConfig(&cfg)

	if cfg.Timeout != 30000 {
		t.Fatalf("expected default timeout 30000, got %d", cfg.Timeout)
	}
	if cfg.KeepAlive != 60000 {
		t.Fatalf("expected default keep alive 60000, got %d", cfg.KeepAlive)
	}
}

func TestUniqueTopicsKeepsFirstOccurrenceOrder(t *testing.T) {
	topics := uniqueTopics([]string{"a", "b", "a", "c", "b"})

	expected := []string{"a", "b", "c"}
	if len(topics) != len(expected) {
		t.Fatalf("expected %d topics, got %d", len(expected), len(topics))
	}
	for i := range expected {
		if topics[i] != expected[i] {
			t.Fatalf("expected topic at index %d to be %s, got %s", i, expected[i], topics[i])
		}
	}
}
