package mqttx

import (
	"strings"
	"testing"
)

func TestSubscribeGrantedError(t *testing.T) {
	tests := []struct {
		name      string
		topic     string
		requested byte
		result    map[string]byte
		wantErr   string
	}{
		{name: "qos1 正常授予", topic: "dronebox/+/status", requested: 1, result: map[string]byte{"dronebox/+/status": 1}},
		{name: "qos0 正常授予", topic: "a/b", requested: 0, result: map[string]byte{"a/b": 0}},
		{name: "QoS 降级不算失败", topic: "a/b", requested: 1, result: map[string]byte{"a/b": 0}},
		{name: "空结果", topic: "a/b", requested: 1, result: map[string]byte{}, wantErr: "empty SUBACK result"},
		{name: "缺少本次订阅的 topic", topic: "a/b", requested: 1, result: map[string]byte{"c/d": 1}, wantErr: "topic=a/b missing from SUBACK"},
		{name: "拒绝订阅", topic: "dronebox/+/status", requested: 1, result: map[string]byte{"dronebox/+/status": 0x80}, wantErr: "granted=0x80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := subscribeGrantedError(tt.topic, tt.requested, tt.result)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("subscribeGrantedError(%v) error = %v, want nil", tt.result, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("subscribeGrantedError(%v) error = nil, want %q", tt.result, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// 批量订阅时一次汇总所有被拒 topic，顺序稳定，避免只看到一个失败原因。
func TestSubscribeGrantedErrorJoinsAllFailures(t *testing.T) {
	result := map[string]byte{
		"topic/c": 1,
		"topic/b": 0x87,
		"topic/a": 0x80,
		"topic/d": 0xA2,
	}

	err := subscribeGrantedError("topic/c", 1, result)
	if err == nil {
		t.Fatal("subscribeGrantedError() error = nil, want aggregated error")
	}

	msg := err.Error()
	for _, want := range []string{
		"subscribe rejected by broker",
		"topic=topic/a granted=0x80",
		"topic=topic/b granted=0x87",
		"topic=topic/d granted=0xa2",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error = %q, want contains %q", msg, want)
		}
	}
	if strings.Contains(msg, "topic/c") {
		t.Fatalf("error = %q, should not contain granted topic", msg)
	}
	// 排序后 topic/a 在 topic/b、topic/d 之前，日志顺序稳定。
	if strings.Index(msg, "topic/a") > strings.Index(msg, "topic/b") {
		t.Fatalf("error = %q, want sorted topic order", msg)
	}
}
