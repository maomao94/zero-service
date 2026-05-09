package drc

import (
	"context"
	"testing"
	"time"

	"zero-service/app/djicloud/internal/config"
)

func newTestManager(t *testing.T, heartbeatTimeout time.Duration) *Manager {
	t.Helper()
	m := NewManager(nil, config.DrcConfig{
		HeartbeatInterval: time.Hour,
		HeartbeatTimeout:  heartbeatTimeout,
	})
	t.Cleanup(m.Close)
	return m
}

func TestNextSeqRequiresAliveState(t *testing.T) {
	m := newTestManager(t, 30*time.Millisecond)
	gatewaySn := "gateway-seq"

	if err := m.Enable(context.Background(), gatewaySn); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	seq, err := m.GetNextSeq(gatewaySn)
	if err != nil || seq != 0 {
		t.Fatalf("GetNextSeq() = (%d, %v), want (0, nil)", seq, err)
	}

	time.Sleep(80 * time.Millisecond)

	seq, err = m.GetNextSeq(gatewaySn)
	if err == nil || seq != 0 {
		t.Fatalf("GetNextSeq() after ttl = (%d, %v), want (0, error)", seq, err)
	}
}

func TestHeartbeatCancelDoesNotDisableCurrentSession(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-session"

	if err := m.Enable(context.Background(), gatewaySn, WithMaxTimeout(20*time.Millisecond)); err != nil {
		t.Fatalf("Enable() first error = %v", err)
	}

	m.expireSession(gatewaySn, "stale-session")

	if !m.IsAlive(gatewaySn) {
		t.Fatal("IsAlive() = false, want true after stale session cleanup")
	}
}
