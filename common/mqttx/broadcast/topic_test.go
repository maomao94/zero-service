package broadcast

import (
	"strings"
	"testing"
)

func TestTopicConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "broadcast_topic_oryx", got: BroadcastTopic("oryx/server"), want: "oryx/server/broadcast"},
		{name: "broadcast_pattern_oryx", got: BroadcastTopicPattern("oryx/server"), want: "oryx/server/broadcast"},
		{name: "broadcast_ack_oryx", got: BroadcastAckTopic("oryx/server", "xyz"), want: "oryx/server/broadcast_reply/xyz"},
		{name: "broadcast_ack_pattern_oryx", got: BroadcastAckTopicPattern("oryx/server"), want: "oryx/server/broadcast_reply/+"},
		{name: "broadcast_topic_iec", got: BroadcastTopic("iec"), want: "iec/broadcast"},
		{name: "broadcast_pattern_iec", got: BroadcastTopicPattern("iec"), want: "iec/broadcast"},
		{name: "broadcast_ack_iec", got: BroadcastAckTopic("iec", "xyz"), want: "iec/broadcast_reply/xyz"},
		{name: "broadcast_ack_pattern_iec", got: BroadcastAckTopicPattern("iec"), want: "iec/broadcast_reply/+"},
		{name: "prefix", got: Prefix("oryx", "server"), want: "oryx/server"},
		{name: "prefix_single", got: Prefix("iec"), want: "iec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

// matchTopicPattern 断言通配订阅模式（支持 "+" 单段通配）能匹配具体主题。
func matchTopicPattern(pattern, topic string) bool {
	pSegments := strings.Split(pattern, "/")
	tSegments := strings.Split(topic, "/")
	if len(pSegments) != len(tSegments) {
		return false
	}
	for i := range pSegments {
		if pSegments[i] == "+" {
			continue
		}
		if pSegments[i] != tSegments[i] {
			return false
		}
	}
	return true
}

func TestTopicPatternsMatchConcreteTopics(t *testing.T) {
	check := func(prefix, instanceID string) {
		if !matchTopicPattern(BroadcastTopicPattern(prefix), BroadcastTopic(prefix)) {
			t.Fatalf("BroadcastTopicPattern(%q) must match BroadcastTopic(%q)", prefix, BroadcastTopic(prefix))
		}
		if !matchTopicPattern(BroadcastAckTopicPattern(prefix), BroadcastAckTopic(prefix, instanceID)) {
			t.Fatalf("BroadcastAckTopicPattern(%q) must match BroadcastAckTopic(%q, %q)", prefix, prefix, instanceID)
		}
	}
	check("oryx/server", "oryx-relay-uid1")
	check("iec", "iec-caller-uid2")
}

func TestConcreteTopicsDifferFromPatterns(t *testing.T) {
	// 广播主题无通配（pattern 与具体同值）；ack 主题的 pattern 必须带 "+" 通配，与具体不同
	if BroadcastTopicPattern("iec") == BroadcastTopic("iec") {
		// 允许同值（无通配），仅确认值正确即可
		if BroadcastTopicPattern("iec") != "iec/broadcast" {
			t.Fatalf("unexpected broadcast topic pattern: %q", BroadcastTopicPattern("iec"))
		}
	}
	if BroadcastAckTopicPattern("iec") == BroadcastAckTopic("iec", "x") {
		t.Fatalf("ack pattern must be a wildcard pattern, got %q", BroadcastAckTopicPattern("iec"))
	}
}
