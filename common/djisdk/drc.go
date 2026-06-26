package djisdk

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DrcConfig struct {
	HeartbeatInterval time.Duration `json:",default=2s"`
	HeartbeatTimeout  time.Duration `json:",default=300s"`
}

const (
	defaultHeartbeatInterval = 2 * time.Second
	defaultHeartbeatTimeout  = 300 * time.Second
)

func DefaultDrcConfig() DrcConfig {
	return DrcConfig{
		HeartbeatInterval: defaultHeartbeatInterval,
		HeartbeatTimeout:  defaultHeartbeatTimeout,
	}
}

func (c DrcConfig) normalized() DrcConfig {
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = defaultHeartbeatInterval
	}
	if c.HeartbeatTimeout <= 0 {
		c.HeartbeatTimeout = defaultHeartbeatTimeout
	}
	return c
}

type DrcSessionExpiredHook  func(gatewaySn, sessionID, reason string)
type DrcSessionEnabledHook  func(gatewaySn, sessionID string)
type DrcSessionDisabledHook func(gatewaySn, sessionID string)

type DrcEnableOption func(*drcEnableOptions)

type drcEnableOptions struct {
	maxTimeout time.Duration
}

func WithDrcMaxTimeout(d time.Duration) DrcEnableOption {
	return func(o *drcEnableOptions) { o.maxTimeout = d }
}

type drcManagerOption func(*drcManager)

func withDrcOnSessionExpired(hook DrcSessionExpiredHook) drcManagerOption {
	return func(m *drcManager) { m.onSessionExpired = hook }
}

func withDrcOnSessionEnabled(hook DrcSessionEnabledHook) drcManagerOption {
	return func(m *drcManager) { m.onSessionEnabled = hook }
}

func withDrcOnSessionDisabled(hook DrcSessionDisabledHook) drcManagerOption {
	return func(m *drcManager) { m.onSessionDisabled = hook }
}

type drcManager struct {
	mu      sync.RWMutex
	session map[string]*drcDeviceSession
	client  *Client
	config  DrcConfig

	onSessionExpired  DrcSessionExpiredHook
	onSessionEnabled  DrcSessionEnabledHook
	onSessionDisabled DrcSessionDisabledHook

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func newDrcManager(client *Client, cfg DrcConfig, opts ...drcManagerOption) *drcManager {
	cfg = cfg.normalized()
	ctx, cancel := context.WithCancel(context.Background())
	m := &drcManager{
		session: make(map[string]*drcDeviceSession),
		client:  client,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
	}
	for _, o := range opts {
		o(m)
	}
	go m.cleanLoop()
	logx.Infof("[drc-manager] initialized: heartbeat_interval=%v heartbeat_timeout=%v", cfg.HeartbeatInterval, cfg.HeartbeatTimeout)
	return m
}

func (m *drcManager) Close() {
	m.closeOnce.Do(func() { m.cancel() })
}

func (m *drcManager) Enable(ctx context.Context, gatewaySn string, maxTimeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.loadOrInitSession(gatewaySn)
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.IsAlive(m.config.HeartbeatTimeout) {
		logx.WithContext(ctx).Debugf("[drc-manager] enable skipped (already alive): gateway_sn=%s session_id=%s", gatewaySn, session.SessionID)
		return nil
	}

	if session.heartbeatCancel != nil {
		logx.WithContext(ctx).Debugf("[drc-manager] stopping stale heartbeat before re-enable: gateway_sn=%s", gatewaySn)
		session.heartbeatCancel()
		session.heartbeatCancel = nil
	}

	session.Enabled = true
	session.StartedAt = time.Now()
	session.UpdateHeartbeat()
	session.SessionID = uuid.New().String()
	session.seq.Store(0)
	if maxTimeout > 0 {
		session.MaxDeadline = session.StartedAt.Add(maxTimeout)
	} else {
		session.MaxDeadline = time.Time{}
	}
	m.session[gatewaySn] = session

	m.startHeartbeat(session)
	logx.WithContext(ctx).Infof("[drc-manager] enabled: gateway_sn=%s session_id=%s max_timeout=%v", gatewaySn, session.SessionID, maxTimeout)
	m.fireSessionEnabled(gatewaySn, session.SessionID)
	return nil
}

func (m *drcManager) Disable(ctx context.Context, gatewaySn string) error {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if !ok {
		logx.WithContext(ctx).Debugf("[drc-manager] disable skipped (already disabled): gateway_sn=%s", gatewaySn)
		return nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.Enabled {
		logx.WithContext(ctx).Debugf("[drc-manager] disable skipped (already disabled): gateway_sn=%s", gatewaySn)
		return nil
	}
	sessionID := session.SessionID
	session.Enabled = false
	m.cancelHeartbeat(session)
	logx.WithContext(ctx).Infof("[drc-manager] disabled: gateway_sn=%s session_id=%s", gatewaySn, sessionID)
	m.fireSessionDisabled(gatewaySn, sessionID)
	return nil
}

func (m *drcManager) OnDeviceHeartbeat(ctx context.Context, gatewaySn string) {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()
	if !ok {
		logx.WithContext(ctx).Infof("[drc-manager] device heartbeat received but state not found (not enabled): %s", gatewaySn)
		return
	}

	session.mu.Lock()
	if !session.Enabled {
		session.mu.Unlock()
		return
	}
	if !session.MaxDeadline.IsZero() && time.Now().After(session.MaxDeadline) {
		sessionID := session.SessionID
		session.Enabled = false
		m.cancelHeartbeat(session)
		session.mu.Unlock()
		m.fireSessionExpired(gatewaySn, sessionID, "max_deadline_exceeded")
		return
	}
	session.UpdateHeartbeat()
	session.mu.Unlock()
	logx.WithContext(ctx).Debugf("[drc-manager] device heartbeat refreshed: gateway_sn=%s", gatewaySn)
}

func (m *drcManager) GetNextSeq(gatewaySn string) (int, error) {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()

	if !ok {
		return 0, status.Errorf(codes.FailedPrecondition,
			"DRC mode not enabled for device=%s, please call DrcModeEnter first", gatewaySn)
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.IsAlive(m.config.HeartbeatTimeout) {
		return 0, status.Errorf(codes.FailedPrecondition,
			"DRC mode not enabled for device=%s, please call DrcModeEnter first", gatewaySn)
	}
	seq := session.seq.Add(1) - 1
	return int(seq), nil
}

func (m *drcManager) GetStatus(gatewaySn string) (enabled bool, startedAt, lastHb time.Time, nextSeq int, alive bool) {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()

	if !ok {
		return false, time.Time{}, time.Time{}, 0, false
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	alive = session.IsAlive(m.config.HeartbeatTimeout)
	enabled = alive
	startedAt = session.StartedAt
	lastHb = session.GetLastHeartbeat()
	nextSeq = int(session.seq.Load())
	return
}

func (m *drcManager) startHeartbeat(session *drcDeviceSession) {
	deadline := session.MaxDeadline

	var heartbeatCtx context.Context
	var heartbeatCancel context.CancelFunc
	if !deadline.IsZero() {
		heartbeatCtx, heartbeatCancel = context.WithDeadline(m.ctx, deadline)
	} else {
		heartbeatCtx, heartbeatCancel = context.WithCancel(m.ctx)
	}
	session.heartbeatCancel = heartbeatCancel

	go m.heartbeatLoop(session.GatewaySn, session.SessionID, heartbeatCtx)
}

// heartbeatLoop 定期经 drc/down 发送 heart_beat，保持 DRC 链路存活。
// 会话过期或为非活跃 session 时自动退出，不重复发心跳。
func (m *drcManager) heartbeatLoop(gatewaySn, sessionID string, heartbeatCtx context.Context) {
	ticker := time.NewTicker(m.config.HeartbeatInterval)
	defer ticker.Stop()

	logx.Debugf("[drc-heartbeat] started: gateway_sn=%s session_id=%s interval=%v", gatewaySn, sessionID, m.config.HeartbeatInterval)

	for {
		select {
		case <-heartbeatCtx.Done():
			logx.Infof("[drc-heartbeat] context done: gateway_sn=%s session_id=%s reason=%v", gatewaySn, sessionID, heartbeatCtx.Err())
			if heartbeatCtx.Err() == context.DeadlineExceeded {
				m.expireSession(gatewaySn, sessionID)
			}
			return
		case <-ticker.C:
			if !m.isCurrentSessionAlive(gatewaySn, sessionID) {
				logx.Infof("[drc-heartbeat] stale session or not alive, stop: gateway_sn=%s session_id=%s", gatewaySn, sessionID)
				return
			}
			if m.client == nil {
				continue
			}
			sendCtx, cancel := context.WithTimeout(heartbeatCtx, m.config.HeartbeatInterval)
			_, err := m.client.SendDrcHeartBeat(sendCtx, gatewaySn, time.Now().UnixMilli())
			cancel()
			if err != nil {
				logx.Errorf("[drc-heartbeat] send failed: gateway_sn=%s err=%v", gatewaySn, err)
			} else {
				logx.Debugf("[drc-heartbeat] sent: gateway_sn=%s", gatewaySn)
			}
		}
	}
}

func (m *drcManager) isCurrentSessionAlive(gatewaySn, sessionID string) bool {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()

	if !ok {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.isCurrentSessionAlive(sessionID, m.config.HeartbeatTimeout)
}

func (m *drcManager) cancelHeartbeat(session *drcDeviceSession) {
	cancel := session.heartbeatCancel
	session.heartbeatCancel = nil
	if cancel != nil {
		cancel()
	}
}

func (m *drcManager) expireSession(gatewaySn, sessionID string) {
	m.mu.RLock()
	session, ok := m.session[gatewaySn]
	m.mu.RUnlock()

	if !ok {
		return
	}
	session.mu.Lock()
	if session.SessionID != sessionID {
		session.mu.Unlock()
		return
	}
	session.Enabled = false
	session.heartbeatCancel = nil
	session.mu.Unlock()
	m.fireSessionExpired(gatewaySn, sessionID, "max_deadline_exceeded")
}

// cleanLoop 定期扫描 session map，清理超时未心跳的过期状态。
func (m *drcManager) cleanLoop() {
	interval := m.config.HeartbeatTimeout / 2
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	if interval > 15*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			count := m.cleanupExpiredStates()
			logx.Debugf("[drc-clean] cleaned %d expired states", count)
		}
	}
}

// cleanupExpiredStates 扫描并删除超时 session，对 enabled 但已过期的 session 触发 onSessionExpired 回调。
func (m *drcManager) cleanupExpiredStates() int {
	expired := make(map[string]*drcDeviceSession)

	count := 0
	m.mu.Lock()
	for gatewaySn, session := range m.session {
		session.mu.Lock()
		alive := session.IsAlive(m.config.HeartbeatTimeout)
		if alive {
			session.mu.Unlock()
			continue
		}
		needNotify := session.Enabled
		session.Enabled = false
		session.mu.Unlock()

		delete(m.session, gatewaySn)
		count++
		if needNotify {
			expired[gatewaySn] = session
		}
	}
	m.mu.Unlock()

	for gw, session := range expired {
		m.cancelHeartbeat(session)
		m.fireSessionExpired(gw, session.SessionID, "heartbeat_timeout")
	}
	return count
}

func (m *drcManager) loadOrInitSession(gatewaySn string) *drcDeviceSession {
	if session, ok := m.session[gatewaySn]; ok {
		return session
	}
	session := &drcDeviceSession{GatewaySn: gatewaySn}
	m.session[gatewaySn] = session
	return session
}

// ==================== hooks ====================

func (m *drcManager) fireSessionExpired(gatewaySn, sessionID, reason string) {
	if m.onSessionExpired != nil {
		threading.GoSafe(func() { m.onSessionExpired(gatewaySn, sessionID, reason) })
	}
}

func (m *drcManager) fireSessionEnabled(gatewaySn, sessionID string) {
	if m.onSessionEnabled != nil {
		threading.GoSafe(func() { m.onSessionEnabled(gatewaySn, sessionID) })
	}
}

func (m *drcManager) fireSessionDisabled(gatewaySn, sessionID string) {
	if m.onSessionDisabled != nil {
		threading.GoSafe(func() { m.onSessionDisabled(gatewaySn, sessionID) })
	}
}

// ==================== DeviceSession ====================

type drcDeviceSession struct {
	GatewaySn       string
	Enabled         bool
	StartedAt       time.Time
	SessionID       string
	MaxDeadline     time.Time
	seq             atomic.Int64
	lastHeartbeat   atomic.Int64
	heartbeatCancel context.CancelFunc
	mu              sync.Mutex
}

func (s *drcDeviceSession) UpdateHeartbeat() {
	s.lastHeartbeat.Store(time.Now().UnixMilli())
}

func (s *drcDeviceSession) GetLastHeartbeat() time.Time {
	ms := s.lastHeartbeat.Load()
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

func (s *drcDeviceSession) IsAlive(heartbeatTimeout time.Duration) bool {
	if !s.Enabled {
		return false
	}
	now := time.Now()
	if !s.MaxDeadline.IsZero() && now.After(s.MaxDeadline) {
		return false
	}
	ms := s.lastHeartbeat.Load()
	if ms == 0 {
		return false
	}
	return now.Sub(time.UnixMilli(ms)) < heartbeatTimeout
}

func (s *drcDeviceSession) isCurrentSessionAlive(sessionID string, heartbeatTimeout time.Duration) bool {
	return s.SessionID == sessionID && s.IsAlive(heartbeatTimeout)
}
