package djisdk

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testDrcConfig 使用大心跳间隔，避免测试期间 ticker 触发；超时使用分钟级，
// 过期场景通过直接改写 lastHeartbeat/MaxDeadline 触发，保证确定性。
func testDrcConfig() DrcConfig {
	return DrcConfig{
		HeartbeatInterval: time.Hour,
		HeartbeatTimeout:  time.Minute,
	}
}

func TestDrcManagerEnableStartsSessionAndFiresHook(t *testing.T) {
	var enabledGW string
	client := NewClient(nil,
		WithDrcSessionEnabled(func(ctx context.Context, gatewaySn, sessionID string) {
			enabledGW = gatewaySn
		}),
	)
	if client.drcManager == nil {
		t.Fatal("drcManager not created")
	}
	defer client.drcManager.Close()

	if err := client.EnableDrc(context.Background(), "dock-1", WithDrcMaxTimeout(30*time.Minute)); err != nil {
		t.Fatalf("EnableDrc() error = %v", err)
	}
	if enabledGW != "dock-1" {
		t.Fatalf("enabled hook gateway = %q", enabledGW)
	}

	enabled, startedAt, lastHb, nextSeq, alive := client.DrcStatus("dock-1")
	if !enabled || !alive || startedAt.IsZero() || lastHb.IsZero() || nextSeq != 0 {
		t.Fatalf("DrcStatus = %v, %v, %v, %d, %v", enabled, startedAt, lastHb, nextSeq, alive)
	}
}

func TestDrcManagerEnableIdempotentWhileAlive(t *testing.T) {
	hookCount := 0
	client := NewClient(nil,
		WithDrcSessionEnabled(func(ctx context.Context, gatewaySn, sessionID string) { hookCount++ }),
	)
	defer client.drcManager.Close()

	if err := client.EnableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("EnableDrc() error = %v", err)
	}
	if err := client.EnableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("second EnableDrc() error = %v", err)
	}
	if hookCount != 1 {
		t.Fatalf("enabled hook fired %d times, want 1", hookCount)
	}
}

func TestDrcNextSeqSequence(t *testing.T) {
	m := newDrcManager(nil, testDrcConfig())
	defer m.Close()

	if _, err := m.GetNextSeq("dock-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("GetNextSeq before enable error = %v, want FailedPrecondition", err)
	}

	if err := m.Enable(context.Background(), "dock-1", 0); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	want := []int{0, 1, 2}
	for _, w := range want {
		seq, err := m.GetNextSeq("dock-1")
		if err != nil || seq != w {
			t.Fatalf("GetNextSeq() = %d, %v, want %d", seq, err, w)
		}
	}
}

func TestDrcStatusBeforeEnable(t *testing.T) {
	client := NewClient(nil)
	defer client.drcManager.Close()

	enabled, startedAt, lastHb, nextSeq, alive := client.DrcStatus("dock-1")
	if enabled || alive || !startedAt.IsZero() || !lastHb.IsZero() || nextSeq != 0 {
		t.Fatalf("DrcStatus = %v, %v, %v, %d, %v", enabled, startedAt, lastHb, nextSeq, alive)
	}
}

func TestDrcDisableFiresHookAndClearsState(t *testing.T) {
	var disabledGW string
	client := NewClient(nil,
		WithDrcSessionDisabled(func(ctx context.Context, gatewaySn, sessionID string) { disabledGW = gatewaySn }),
	)
	defer client.drcManager.Close()

	if err := client.EnableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("EnableDrc() error = %v", err)
	}
	if err := client.DisableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("DisableDrc() error = %v", err)
	}
	if disabledGW != "dock-1" {
		t.Fatalf("disabled hook gateway = %q", disabledGW)
	}

	enabled, _, _, _, alive := client.DrcStatus("dock-1")
	if enabled || alive {
		t.Fatalf("DrcStatus after disable = enabled=%v alive=%v", enabled, alive)
	}

	if err := client.DisableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("second DisableDrc() error = %v", err)
	}
}

func TestDrcHeartbeatRefreshesSession(t *testing.T) {
	m := newDrcManager(nil, testDrcConfig())
	defer m.Close()

	if err := m.Enable(context.Background(), "dock-1", 0); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	session := m.session["dock-1"]
	old := session.GetLastHeartbeat()

	time.Sleep(2 * time.Millisecond)
	m.OnDeviceHeartbeat(context.Background(), "dock-1")
	if got := session.GetLastHeartbeat(); !got.After(old) {
		t.Fatalf("last heartbeat = %v, want after %v", got, old)
	}

	session.lastHeartbeat.Store(time.Now().Add(-2 * time.Minute).UnixMilli())
	if _, err := m.GetNextSeq("dock-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("GetNextSeq after stale heartbeat error = %v, want FailedPrecondition", err)
	}
}

func TestDrcCleanupRemovesExpiredSessionAndFiresExpired(t *testing.T) {
	var expiredGW, reason string
	m := newDrcManager(nil, testDrcConfig(),
		withDrcOnSessionExpired(func(ctx context.Context, gatewaySn, sessionID, r string) {
			expiredGW, reason = gatewaySn, r
		}),
	)
	defer m.Close()

	if err := m.Enable(context.Background(), "dock-1", 0); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	m.session["dock-1"].lastHeartbeat.Store(time.Now().Add(-2 * time.Minute).UnixMilli())

	if count := m.cleanupExpiredStates(); count != 1 {
		t.Fatalf("cleanup count = %d, want 1", count)
	}
	if _, ok := m.session["dock-1"]; ok {
		t.Fatal("expired session still present")
	}
	if expiredGW != "dock-1" || reason != "heartbeat_timeout" {
		t.Fatalf("expired hook = %q, %q", expiredGW, reason)
	}
}

func TestDrcMaxDeadlineExpiryOnHeartbeat(t *testing.T) {
	var reason string
	m := newDrcManager(nil, testDrcConfig(),
		withDrcOnSessionExpired(func(ctx context.Context, gatewaySn, sessionID, r string) { reason = r }),
	)
	defer m.Close()

	if err := m.Enable(context.Background(), "dock-1", time.Hour); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	m.session["dock-1"].MaxDeadline = time.Now().Add(-time.Second)

	m.OnDeviceHeartbeat(context.Background(), "dock-1")
	if reason != "max_deadline_exceeded" {
		t.Fatalf("expired reason = %q", reason)
	}
	if _, err := m.GetNextSeq("dock-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("GetNextSeq after deadline error = %v, want FailedPrecondition", err)
	}
}

func TestDrcSessionReplacementResetsSeqAndID(t *testing.T) {
	client := NewClient(nil)
	defer client.drcManager.Close()

	if err := client.EnableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("EnableDrc() error = %v", err)
	}
	sessionID := client.drcManager.session["dock-1"].SessionID
	if seq, err := client.DrcNextSeq("dock-1"); err != nil || seq != 0 {
		t.Fatalf("DrcNextSeq() = %d, %v", seq, err)
	}

	if err := client.DisableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("DisableDrc() error = %v", err)
	}
	if err := client.EnableDrc(context.Background(), "dock-1"); err != nil {
		t.Fatalf("second EnableDrc() error = %v", err)
	}

	session := client.drcManager.session["dock-1"]
	if session.SessionID == sessionID {
		t.Fatal("session ID was not replaced after re-enable")
	}
	if seq, err := client.DrcNextSeq("dock-1"); err != nil || seq != 0 {
		t.Fatalf("DrcNextSeq() after re-enable = %d, %v, want 0", seq, err)
	}
}

func TestClientDrcAPIsWithoutManager(t *testing.T) {
	client := buildClient(nil, &clientOptions{pendingTTL: time.Second})
	if client.drcManager != nil {
		t.Fatal("drcManager should be nil when drcConfig is zero")
	}

	if err := client.EnableDrc(context.Background(), "dock-1"); err == nil || !strings.Contains(err.Error(), "drcManager not configured") {
		t.Fatalf("EnableDrc() error = %v", err)
	}
	if _, err := client.DrcNextSeq("dock-1"); err == nil || !strings.Contains(err.Error(), "drcManager not configured") {
		t.Fatalf("DrcNextSeq() error = %v", err)
	}
	enabled, startedAt, lastHb, nextSeq, alive := client.DrcStatus("dock-1")
	if enabled || alive || !startedAt.IsZero() || !lastHb.IsZero() || nextSeq != 0 {
		t.Fatalf("DrcStatus = %v, %v, %v, %d, %v", enabled, startedAt, lastHb, nextSeq, alive)
	}
}
