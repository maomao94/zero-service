package djisdk

import (
	"context"
	"testing"
	"time"
)

func TestDefaultReplyConfigDisablesReplies(t *testing.T) {
	cfg := DefaultReplyConfig()
	if cfg.EnableEventReply || cfg.EnableStatusReply || cfg.EnableRequestReply {
		t.Fatalf("DefaultReplyConfig() = %+v, want all replies disabled by default", cfg)
	}
}

func TestWithPendingTTLIgnoresZero(t *testing.T) {
	opt := defaultClientOptions()
	WithPendingTTL(0)(&opt)
	if opt.pendingTTL != defaultPendingTTL {
		t.Fatalf("pendingTTL = %v, want default %v", opt.pendingTTL, defaultPendingTTL)
	}

	WithPendingTTL(5 * time.Second)(&opt)
	if opt.pendingTTL != 5*time.Second {
		t.Fatalf("pendingTTL = %v, want 5s", opt.pendingTTL)
	}
}

func TestWithReplyConfigOverridesDefault(t *testing.T) {
	cfg := ReplyConfig{EnableEventReply: true, EnableStatusReply: false, EnableRequestReply: true}
	client := NewClient(nil, WithReplyConfig(cfg))
	if client.reply != cfg {
		t.Fatalf("reply = %+v, want %+v", client.reply, cfg)
	}
}

func TestHandlerOptionsRegisterFields(t *testing.T) {
	called := false
	handler := func(ctx context.Context, gatewaySn string, data *OsdMessage) error { called = true; return nil }

	client := NewClient(nil, WithOsdHandler(handler))
	if client.handlers.onOsd == nil {
		t.Fatal("onOsd handler not registered")
	}
	if err := client.handlers.onOsd(context.Background(), "dock-1", &OsdMessage{}); err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if !called {
		t.Fatal("registered handler was not invoked")
	}
}

func TestWithOnlineCheckerRegistersChecker(t *testing.T) {
	checker := func(gatewaySn string) bool { return gatewaySn == "online-1" }
	client := NewClient(nil, WithOnlineChecker(checker))
	if client.handlers.onlineChecker == nil {
		t.Fatal("onlineChecker not registered")
	}
	if !client.handlers.onlineChecker("online-1") || client.handlers.onlineChecker("offline-1") {
		t.Fatal("onlineChecker returned unexpected result")
	}
}

func TestDefaultClientOptionsUsesSanitizedDrcConfig(t *testing.T) {
	opt := defaultClientOptions()
	if opt.reply != DefaultReplyConfig() {
		t.Fatalf("reply = %+v", opt.reply)
	}
	if opt.drcConfig.HeartbeatInterval <= 0 || opt.drcConfig.HeartbeatTimeout <= 0 {
		t.Fatalf("drcConfig = %+v, want nonzero durations", opt.drcConfig)
	}
}

func TestDefaultDrcConfigNormalized(t *testing.T) {
	cfg := DefaultDrcConfig()
	if cfg.HeartbeatInterval != defaultHeartbeatInterval || cfg.HeartbeatTimeout != defaultHeartbeatTimeout {
		t.Fatalf("DefaultDrcConfig() = %+v", cfg)
	}

	normalized := (DrcConfig{}).normalized()
	if normalized.HeartbeatInterval != defaultHeartbeatInterval || normalized.HeartbeatTimeout != defaultHeartbeatTimeout {
		t.Fatalf("normalized zero config = %+v", normalized)
	}
}
