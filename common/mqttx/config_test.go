package mqttx

import (
	"context"
	"testing"
)

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

func TestAdjustConfigKeepsExplicitValuesAndClampsQos(t *testing.T) {
	cfg := MqttConfig{Timeout: 1000, KeepAlive: 2000, Qos: 3}
	adjustConfig(&cfg)
	if cfg.Timeout != 1000 {
		t.Fatalf("expected timeout 1000 to be preserved, got %d", cfg.Timeout)
	}
	if cfg.KeepAlive != 2000 {
		t.Fatalf("expected keep alive 2000 to be preserved, got %d", cfg.KeepAlive)
	}
	if cfg.Qos != 1 {
		t.Fatalf("expected qos 3 to be clamped to 1, got %d", cfg.Qos)
	}

	cfg = MqttConfig{Qos: 2}
	adjustConfig(&cfg)
	if cfg.Qos != 2 {
		t.Fatalf("expected valid qos 2 to be preserved, got %d", cfg.Qos)
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

func TestWithOnReadySetsCallback(t *testing.T) {
	options := &ClientOptions{}
	var got Client
	WithOnReady(func(client Client) { got = client })(options)
	if options.onReady == nil {
		t.Fatal("expected onReady callback to be registered")
	}

	sentinel := &mqttClient{cfg: MqttConfig{ClientID: "sentinel"}}
	options.onReady(sentinel)
	if got != sentinel {
		t.Fatalf("onReady received %v, want the sentinel client", got)
	}
}

func TestWithReplyRouterRegistersNonNilRouter(t *testing.T) {
	router := NewReplyRouter[string](ReplyDecoderFunc[string](func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	}))
	defer router.Close()

	t.Run("nil router is ignored", func(t *testing.T) {
		options := &ClientOptions{}
		WithReplyRouter[string]("reply/+", nil)(options)
		if len(options.replyRouters) != 0 {
			t.Fatalf("registered routers = %d, want 0", len(options.replyRouters))
		}
	})

	t.Run("router is registered with template", func(t *testing.T) {
		options := &ClientOptions{}
		WithReplyRouter("reply/+", router)(options)
		if len(options.replyRouters) != 1 {
			t.Fatalf("registered routers = %d, want 1", len(options.replyRouters))
		}
		if options.replyRouters[0].topicTemplate != "reply/+" || options.replyRouters[0].handler != router {
			t.Fatalf("unexpected registration: %+v", options.replyRouters[0])
		}
	})
}
