package drc

import (
	"context"
	"sync"
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

func TestExpireSessionDoesNotAffectNewerSession(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-expire-cache-miss"

	// Enable a session
	if err := m.Enable(context.Background(), gatewaySn); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	// Get the session
	m.mu.RLock()
	session := m.session[gatewaySn]
	m.mu.RUnlock()

	// Simulate an old session trying to expire
	m.expireSession(gatewaySn, "old-session-id")

	// Current session should still be alive
	if !session.Enabled {
		t.Fatal("expireSession() disabled newer session")
	}
}

func TestCleanupExpiredStatesKeepsAliveState(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-alive"
	session := &DeviceSession{
		GatewaySn: gatewaySn,
		Enabled:   true,
		SessionID: "alive-session",
		StartedAt: time.Now(),
	}
	session.UpdateHeartbeat()

	m.mu.Lock()
	m.session[gatewaySn] = session
	m.mu.Unlock()

	m.cleanupExpiredStates()

	m.mu.RLock()
	_, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if !ok {
		t.Fatal("cleanupExpiredStates() removed alive session")
	}
}

func TestCleanupExpiredStatesRemovesExpiredState(t *testing.T) {
	m := newTestManager(t, 30*time.Millisecond)
	gatewaySn := "gateway-expired"
	session := &DeviceSession{
		GatewaySn: gatewaySn,
		Enabled:   true,
		SessionID: "expired-session",
		StartedAt: time.Now().Add(-time.Second),
	}
	// 设置心跳为 1 秒前，已经超过 30ms 的超时时间
	session.lastHeartbeat.Store(time.Now().Add(-time.Second).UnixMilli())

	m.mu.Lock()
	m.session[gatewaySn] = session
	m.mu.Unlock()

	m.cleanupExpiredStates()

	m.mu.RLock()
	_, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if ok {
		t.Fatal("cleanupExpiredStates() kept expired session")
	}
}

func TestCleanupExpiredStatesRemovesDisabledState(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-disabled"
	session := &DeviceSession{
		GatewaySn: gatewaySn,
		Enabled:   false,
		SessionID: "disabled-session",
		StartedAt: time.Now(),
	}
	session.UpdateHeartbeat()

	m.mu.Lock()
	m.session[gatewaySn] = session
	m.mu.Unlock()

	m.cleanupExpiredStates()

	m.mu.RLock()
	_, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if ok {
		t.Fatal("cleanupExpiredStates() kept disabled session")
	}
}

func TestCleanupExpiredStatesFiresExpiredHook(t *testing.T) {
	expired := make(chan string, 1)
	m := NewManager(nil, config.DrcConfig{
		HeartbeatInterval: time.Hour,
		HeartbeatTimeout:  30 * time.Millisecond,
	}, WithOnSessionExpired(func(_, sessionID, reason string) {
		expired <- sessionID + ":" + reason
	}))
	t.Cleanup(m.Close)
	gatewaySn := "gateway-expired-hook"
	session := &DeviceSession{
		GatewaySn: gatewaySn,
		Enabled:   true,
		SessionID: "expired-session",
		StartedAt: time.Now().Add(-time.Second),
	}
	session.lastHeartbeat.Store(time.Now().Add(-time.Second).UnixMilli())

	m.mu.Lock()
	m.session[gatewaySn] = session
	m.mu.Unlock()

	m.cleanupExpiredStates()

	select {
	case got := <-expired:
		want := "expired-session:heartbeat_timeout"
		if got != want {
			t.Fatalf("expired hook = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("expired hook was not called")
	}
}

func TestCleanupExpiredStatesDoesNotFireHookForDisabledState(t *testing.T) {
	expired := make(chan string, 1)
	m := NewManager(nil, config.DrcConfig{
		HeartbeatInterval: time.Hour,
		HeartbeatTimeout:  time.Second,
	}, WithOnSessionExpired(func(_, sessionID, reason string) {
		expired <- sessionID + ":" + reason
	}))
	t.Cleanup(m.Close)
	gatewaySn := "gateway-disabled-hook"
	session := &DeviceSession{
		GatewaySn: gatewaySn,
		Enabled:   false,
		SessionID: "disabled-session",
		StartedAt: time.Now(),
	}
	session.UpdateHeartbeat()

	m.mu.Lock()
	m.session[gatewaySn] = session
	m.mu.Unlock()

	m.cleanupExpiredStates()

	select {
	case got := <-expired:
		t.Fatalf("expired hook was called for disabled session: %q", got)
	case <-time.After(100 * time.Millisecond):
		// expected: disabled session does not trigger expired hook
	}
}

func TestDisableStopsHeartbeatGoroutineImmediately(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-disable-stop"

	if err := m.Enable(context.Background(), gatewaySn); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	m.mu.RLock()
	session := m.session[gatewaySn]
	m.mu.RUnlock()

	session.mu.Lock()
	cancel := session.heartbeatCancel
	session.mu.Unlock()
	if cancel == nil {
		t.Fatal("heartbeatCancel should be set after Enable")
	}

	if err := m.Disable(context.Background(), gatewaySn); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	session.mu.Lock()
	cancelAfter := session.heartbeatCancel
	session.mu.Unlock()
	if cancelAfter != nil {
		t.Fatal("heartbeatCancel should be nil after Disable")
	}
}

func TestConcurrentDisableAndOnDeviceHeartbeat(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-concurrent"

	if err := m.Enable(context.Background(), gatewaySn); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.OnDeviceHeartbeat(context.Background(), gatewaySn)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		_ = m.Disable(context.Background(), gatewaySn)
	}()

	wg.Wait()

	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if ok {
		session.mu.Lock()
		alive := session.IsAlive(m.config.HeartbeatTimeout)
		session.mu.Unlock()
		if alive {
			t.Fatal("session should not be alive after Disable")
		}
	}
}

func TestConcurrentEnableDisableOnDeviceHeartbeat(t *testing.T) {
	m := newTestManager(t, time.Second)
	gatewaySn := "gateway-stress"

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = m.Enable(context.Background(), gatewaySn)
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = m.Disable(context.Background(), gatewaySn)
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.OnDeviceHeartbeat(context.Background(), gatewaySn)
		}
	}()

	wg.Wait()
}
