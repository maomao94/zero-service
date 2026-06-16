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

func TestExpireSessionCacheMissDoesNotDeleteNewerWorker(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-expire-cache-miss"
	currentCanceled := false
	currentWorker := &heartbeatWorker{sessionID: "current-session", cancel: func() { currentCanceled = true }}
	m.workers.Store(gatewaySn, currentWorker)

	m.expireSession(gatewaySn, "old-session")

	if currentCanceled {
		t.Fatal("expireSession() canceled newer worker after cache miss")
	}
	val, ok := m.workers.Load(gatewaySn)
	if !ok {
		t.Fatal("expireSession() deleted newer worker after cache miss")
	}
	if val != currentWorker {
		t.Fatalf("workers.Load() = %p, want current worker %p", val, currentWorker)
	}
}

func TestHeartbeatWorkerCleanupDoesNotDeleteCurrentWorker(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-worker"
	oldWorker := &heartbeatWorker{sessionID: "old", cancel: func() {}}
	currentWorker := &heartbeatWorker{sessionID: "current", cancel: func() {}}

	m.workers.Store(gatewaySn, currentWorker)

	if m.deleteHeartbeatWorkerIfCurrent(gatewaySn, oldWorker) {
		t.Fatal("deleteHeartbeatWorkerIfCurrent() = true for stale worker, want false")
	}
	val, ok := m.workers.Load(gatewaySn)
	if !ok {
		t.Fatal("workers.Load() missing current worker after stale cleanup")
	}
	if val != currentWorker {
		t.Fatalf("workers.Load() = %p, want current worker %p", val, currentWorker)
	}

	if !m.deleteHeartbeatWorkerIfCurrent(gatewaySn, currentWorker) {
		t.Fatal("deleteHeartbeatWorkerIfCurrent() = false for current worker, want true")
	}
}

func TestCleanupHeartbeatWorkerKeepsAliveWorker(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-alive"
	now := time.Now()
	canceled := false
	worker := &heartbeatWorker{sessionID: "alive-session", cancel: func() { canceled = true }}
	state := &State{
		GatewaySn:           gatewaySn,
		Enabled:             true,
		SessionID:           worker.sessionID,
		StartedAt:           now,
		LastDeviceHeartbeat: now,
	}
	m.cache.Set(gatewaySn, state)
	m.workers.Store(gatewaySn, worker)

	m.cleanupHeartbeatWorker(gatewaySn, worker, now)

	if canceled {
		t.Fatal("cleanupHeartbeatWorker() canceled alive worker")
	}
	if _, ok := m.workers.Load(gatewaySn); !ok {
		t.Fatal("cleanupHeartbeatWorker() deleted alive worker")
	}
}

func TestCleanupHeartbeatWorkerRemovesExpiredWorker(t *testing.T) {
	m := newTestManager(t, 30*time.Millisecond)
	gatewaySn := "gateway-expired"
	now := time.Now()
	canceled := false
	worker := &heartbeatWorker{sessionID: "expired-session", cancel: func() { canceled = true }}
	state := &State{
		GatewaySn:           gatewaySn,
		Enabled:             true,
		SessionID:           worker.sessionID,
		StartedAt:           now.Add(-time.Second),
		LastDeviceHeartbeat: now.Add(-time.Second),
	}
	m.cache.Set(gatewaySn, state)
	m.workers.Store(gatewaySn, worker)

	m.cleanupHeartbeatWorker(gatewaySn, worker, now)

	if !canceled {
		t.Fatal("cleanupHeartbeatWorker() did not cancel expired worker")
	}
	if _, ok := m.workers.Load(gatewaySn); ok {
		t.Fatal("cleanupHeartbeatWorker() kept expired worker")
	}
	if _, ok := m.cache.Get(gatewaySn); ok {
		t.Fatal("cleanupHeartbeatWorker() kept expired cache entry")
	}
}

func TestCleanupHeartbeatWorkerRemovesDisabledCacheEntry(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-disabled"
	now := time.Now()
	canceled := false
	worker := &heartbeatWorker{sessionID: "disabled-session", cancel: func() { canceled = true }}
	state := &State{
		GatewaySn:           gatewaySn,
		Enabled:             false,
		SessionID:           worker.sessionID,
		StartedAt:           now,
		LastDeviceHeartbeat: now,
	}
	m.cache.Set(gatewaySn, state)
	m.workers.Store(gatewaySn, worker)

	m.cleanupHeartbeatWorker(gatewaySn, worker, now)

	if !canceled {
		t.Fatal("cleanupHeartbeatWorker() did not cancel disabled worker")
	}
	if _, ok := m.workers.Load(gatewaySn); ok {
		t.Fatal("cleanupHeartbeatWorker() kept disabled worker")
	}
	if _, ok := m.cache.Get(gatewaySn); ok {
		t.Fatal("cleanupHeartbeatWorker() kept disabled cache entry")
	}
}

func TestCleanupHeartbeatWorkerDoesNotCancelNewerWorker(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-newer"
	now := time.Now()
	oldCanceled := false
	currentCanceled := false
	oldWorker := &heartbeatWorker{sessionID: "old-session", cancel: func() { oldCanceled = true }}
	currentWorker := &heartbeatWorker{sessionID: "current-session", cancel: func() { currentCanceled = true }}
	state := &State{
		GatewaySn:           gatewaySn,
		Enabled:             true,
		SessionID:           currentWorker.sessionID,
		StartedAt:           now,
		LastDeviceHeartbeat: now,
	}
	m.cache.Set(gatewaySn, state)
	m.workers.Store(gatewaySn, currentWorker)

	m.cleanupHeartbeatWorker(gatewaySn, oldWorker, now)

	if !oldCanceled {
		t.Fatal("cleanupHeartbeatWorker() did not cancel stale worker")
	}
	if currentCanceled {
		t.Fatal("cleanupHeartbeatWorker() canceled newer worker")
	}
	val, ok := m.workers.Load(gatewaySn)
	if !ok {
		t.Fatal("cleanupHeartbeatWorker() deleted newer worker")
	}
	if val != currentWorker {
		t.Fatalf("workers.Load() = %p, want current worker %p", val, currentWorker)
	}
}

func TestCleanupHeartbeatWorkerFiresExpiredHookForCacheMiss(t *testing.T) {
	expired := make(chan string, 1)
	m := NewManager(nil, config.DrcConfig{
		HeartbeatInterval: time.Hour,
		HeartbeatTimeout:  time.Second,
	}, WithOnSessionExpired(func(_, sessionID, reason string) {
		expired <- sessionID + ":" + reason
	}))
	t.Cleanup(m.Close)
	gatewaySn := "gateway-cache-miss"
	worker := &heartbeatWorker{sessionID: "missing-session", cancel: func() {}}
	m.workers.Store(gatewaySn, worker)

	m.cleanupHeartbeatWorker(gatewaySn, worker, time.Now())

	select {
	case got := <-expired:
		want := "missing-session:heartbeat_timeout"
		if got != want {
			t.Fatalf("expired hook = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("expired hook was not called")
	}
}
