package mqttx

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTopicLogConfigShouldLogRateLimits(t *testing.T) {
	conf := newTopicLogConfig(time.Hour)

	if !conf.shouldLog() {
		t.Fatal("first log must be allowed")
	}
	if conf.shouldLog() {
		t.Fatal("second log within the interval must be suppressed")
	}

	atomic.StoreInt64(&conf.lastLogUnix, time.Now().Add(-2*time.Hour).UnixNano())
	if !conf.shouldLog() {
		t.Fatal("log must be allowed after the interval elapsed")
	}
}

func TestTopicLogConfigShouldLogIsAtomic(t *testing.T) {
	conf := newTopicLogConfig(time.Hour)

	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if conf.shouldLog() {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := allowed.Load(); got != 1 {
		t.Fatalf("allowed logs = %d, want exactly 1", got)
	}
}

func TestNewTopicLogConfigFallsBackToDefaultInterval(t *testing.T) {
	if got := newTopicLogConfig(0).minLogInterval; got != 5*time.Second {
		t.Fatalf("min log interval = %v, want 5s", got)
	}
	if got := newTopicLogConfig(time.Minute).minLogInterval; got != time.Minute {
		t.Fatalf("min log interval = %v, want 1m", got)
	}
}

func TestTopicLogManagerDefaultsAndOverrides(t *testing.T) {
	manager := NewTopicLogManager()

	if manager.ShouldLogPayload("device/+") {
		t.Fatal("payload logging must be disabled by default")
	}
	if !manager.ShouldLog("device/+") {
		t.Fatal("first log must be allowed by default")
	}
	if manager.ShouldLog("device/+") {
		t.Fatal("second log must be suppressed by default interval")
	}

	manager.SetLogPayload("device/+", true)
	if !manager.ShouldLogPayload("device/+") {
		t.Fatal("expected payload logging to be enabled for device/+")
	}
	if manager.ShouldLogPayload("other/+") {
		t.Fatal("payload logging must stay isolated per topic")
	}

	manager.SetLogPayload("device/+", false)
	if manager.ShouldLogPayload("device/+") {
		t.Fatal("expected payload logging to be disabled again")
	}
}

func TestTopicLogManagerOptionsApplyToNewTopics(t *testing.T) {
	manager := NewTopicLogManager(
		WithDefaultInterval(time.Hour),
		WithDefaultLogPayload(true),
	)

	if !manager.ShouldLogPayload("device/+") {
		t.Fatal("expected default payload logging to be enabled")
	}
	if !manager.ShouldLog("device/+") {
		t.Fatal("first log must be allowed")
	}
	if manager.ShouldLog("device/+") {
		t.Fatal("expected the configured interval to suppress the second log")
	}
}

func TestTopicLogManagerLoadFromConfig(t *testing.T) {
	manager := NewTopicLogManager()
	manager.LoadFromConfig(TopicLogConfig{
		DefaultLogPayload: true,
		TopicSettings: []TopicLogSetting{
			{TopicTemplate: "device/+", LogPayload: false, MinLogInterval: time.Hour},
			{TopicTemplate: "other/+"},
		},
	})

	if !manager.ShouldLogPayload("unknown/+") {
		t.Fatal("expected default payload logging to apply to unknown topics")
	}
	if manager.ShouldLogPayload("device/+") {
		t.Fatal("expected per-topic payload setting to override the default")
	}
	if manager.ShouldLogPayload("other/+") {
		t.Fatal("per-topic payload setting must override the default even when false")
	}
	if got := manager.getConfig("other/+").minLogInterval; got != 5*time.Second {
		t.Fatalf("zero per-topic interval = %v, want 5s fallback", got)
	}
	if got := manager.getConfig("device/+").minLogInterval; got != time.Hour {
		t.Fatalf("per-topic interval = %v, want 1h", got)
	}

	if !manager.ShouldLog("device/+") {
		t.Fatal("first log must be allowed")
	}
	if manager.ShouldLog("device/+") {
		t.Fatal("expected per-topic interval to suppress the second log")
	}

	// 重新加载应覆盖已有 topic 配置。
	manager.LoadFromConfig(TopicLogConfig{
		TopicSettings: []TopicLogSetting{
			{TopicTemplate: "device/+", LogPayload: true},
		},
	})
	if !manager.ShouldLogPayload("device/+") {
		t.Fatal("expected reload to replace the existing topic config")
	}
	if manager.ShouldLogPayload("fresh/+") {
		t.Fatal("expected reload to reset the default payload setting for new topics")
	}
}
