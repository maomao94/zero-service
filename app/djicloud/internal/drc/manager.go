package drc

import (
	"context"
	"sync"
	"time"
	"zero-service/common/tool"

	"zero-service/app/djicloud/internal/config"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionExpiredHook 会话过期回调函数签名。
// 当 DRC 会话因超时/清理被自动清除时调用，reason 描述清除原因。
// 实现方可在此闭包中发送 Socket 推送事件等通知逻辑。
type SessionExpiredHook func(gatewaySn, sessionID, reason string)

// SessionEnabledHook 会话启用回调函数签名。
// 当 DRC 模式通过 Enable 成功启用时调用（幂等跳过不触发）。
type SessionEnabledHook func(gatewaySn, sessionID string)

// SessionDisabledHook 会话停用回调函数签名。
// 当 DRC 模式通过 Disable 成功停用时调用（幂等跳过不触发）。前端可据此关闭 DRC 按钮。
type SessionDisabledHook func(gatewaySn, sessionID string)

// ManagerOption 配置 Manager 的函数选项。
type ManagerOption func(*Manager)

// WithOnSessionExpired 注册会话过期回调钩子。
// 当 DRC 会话因 MaxDeadline 到期、设备心跳超时、cleanLoop 孤儿清理等原因被自动清除时，
// 会在清除完成后异步调用此 hook。主动调用 Disable 不触发。
func WithOnSessionExpired(hook SessionExpiredHook) ManagerOption {
	return func(m *Manager) {
		m.onSessionExpired = hook
	}
}

// WithOnSessionEnabled 注册会话启用回调钩子。
func WithOnSessionEnabled(hook SessionEnabledHook) ManagerOption {
	return func(m *Manager) {
		m.onSessionEnabled = hook
	}
}

// WithOnSessionDisabled 注册会话停用回调钩子。
func WithOnSessionDisabled(hook SessionDisabledHook) ManagerOption {
	return func(m *Manager) {
		m.onSessionDisabled = hook
	}
}

// Manager DRC 状态管理器，负责所有设备 DRC 会话的生命周期管理与心跳调度。
//
// 并发模型：
//   - m.mu(RWMutex) 保护 session map；写锁仅在 Enable（插入）和 cleanLoop（删除）中持有
//   - DeviceSession.mu 保护单个设备的状态机字段读写（Enabled、SessionID、MaxDeadline 等）
//   - DeviceSession.seq 和 lastHeartbeat 使用 atomic 独立保护
//   - heartbeatCancel 在 Enable 时设置，Disable/expire 时调用并置 nil
//
// 会话清除采用 mark-and-sweep 模式：
//   - Disable/OnDeviceHeartbeat/expireSession 仅标记 Enabled=false 并停止心跳（标记）
//   - cleanLoop 定期扫描 map，统一移除 !IsAlive 的条目（清扫）
//   - 好处：热路径（OnDeviceHeartbeat）只需 RLock，减少写锁竞争
//
// 设备心跳超时判断：
//   - 收到设备心跳上行时更新 lastHeartbeat（atomic.Int64 存 UnixMilli）
//   - cleanLoop 定期扫描，通过 time.Since(lastHeartbeat) > HeartbeatTimeout 判断设备离线
//
// 心跳下发 goroutine 使用 context.Context 控制生命周期：
//   - Enable 时构造子 ctx（含最大超时 deadline 或普通 cancel）
//   - 心跳循环通过 ctx.Done() 监听退出信号
//   - 全局 cleanLoop 定期扫描清理过期会话
type Manager struct {
	mu sync.RWMutex
	// session 设备 DRC 会话表，key=gatewaySn, value=*DeviceSession。
	// 条目存在且 IsAlive() 为 true 表示设备 DRC 存活。
	session   map[string]*DeviceSession
	djiClient *djisdk.Client
	config    config.DrcConfig

	// onSessionExpired 会话过期回调，nil 表示不通知。
	// 仅在非主动 Disable 的清除路径触发（MaxDeadline 到期、心跳超时、cleanLoop 孤儿清理）。
	onSessionExpired SessionExpiredHook

	// onSessionEnabled 会话启用回调，nil 表示不通知。
	// 仅在 Enable 成功（非幂等跳过）时触发。
	onSessionEnabled SessionEnabledHook

	// onSessionDisabled 会话停用回调，nil 表示不通知。
	// 仅在 Disable 成功（非幂等跳过）时触发。前端可据此关闭 DRC 按钮。
	onSessionDisabled SessionDisabledHook

	// 全局 ctx，用于派生所有心跳子 ctx 和 clean goroutine
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// EnableOption 配置 Enable 行为的函数选项。
type EnableOption func(*enableOptions)

type enableOptions struct {
	maxTimeout time.Duration
}

// WithMaxTimeout 设置 DRC 会话最大超时时间。
// 从 Enable 调用时开始计时，到期后强制清除缓存并停止心跳，不受设备心跳续期影响。
// 零值或不传此选项表示无绝对截止时间（设备心跳可无限续期）。
func WithMaxTimeout(d time.Duration) EnableOption {
	return func(o *enableOptions) {
		o.maxTimeout = d
	}
}

// NewManager 创建 DRC 状态管理器并启动后台 cleanLoop。
func NewManager(client *djisdk.Client, cfg config.DrcConfig, opts ...ManagerOption) *Manager {
	ctx, mCancel := context.WithCancel(context.Background())
	m := &Manager{
		session:   make(map[string]*DeviceSession),
		djiClient: client,
		config:    cfg,
		ctx:       ctx,
		cancel:    mCancel,
	}
	for _, o := range opts {
		o(m)
	}
	go m.cleanLoop()
	logx.Infof("[drc-manager] initialized: heartbeat_interval=%v heartbeat_timeout=%v", cfg.HeartbeatInterval, cfg.HeartbeatTimeout)
	return m
}

// Enable 启用设备的 DRC 模式并启动心跳 goroutine。
//
// 幂等性：设备已启用且未过期时直接返回 nil，不会重复启动 goroutine。
// 若设备已启用但心跳已过期（stale 状态），会先停止旧 goroutine 再重新初始化。
// opts 支持 WithMaxTimeout 等可选参数。
func (m *Manager) Enable(ctx context.Context, gatewaySn string, opts ...EnableOption) error {
	var opt enableOptions
	for _, o := range opts {
		o(&opt)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.loadOrInitSession(gatewaySn)
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.IsAlive(m.config.HeartbeatTimeout) {
		logx.WithContext(ctx).Debugf("[drc-manager] enable skipped (already alive): gateway_sn=%s session_id=%s", gatewaySn, session.SessionID)
		return nil
	}

	// 停止旧的心跳 goroutine
	if session.heartbeatCancel != nil {
		logx.WithContext(ctx).Debugf("[drc-manager] stopping stale heartbeat before re-enable: gateway_sn=%s", gatewaySn)
		session.heartbeatCancel()
		session.heartbeatCancel = nil
	}

	session.Enabled = true
	session.StartedAt = time.Now()
	session.UpdateHeartbeat()
	sessionId, _ := tool.SimpleUUID()
	session.SessionID = sessionId
	session.seq.Store(0)
	if opt.maxTimeout > 0 {
		session.MaxDeadline = session.StartedAt.Add(opt.maxTimeout)
	} else {
		session.MaxDeadline = time.Time{}
	}
	sessionID := session.SessionID
	m.session[gatewaySn] = session

	m.startHeartbeat(session)
	logx.WithContext(ctx).Infof("[drc-manager] enabled: gateway_sn=%s session_id=%s max_timeout=%v", gatewaySn, sessionID, opt.maxTimeout)
	m.fireSessionEnabled(gatewaySn, sessionID)
	return nil
}

// Disable 停用设备的 DRC 模式并停止心跳 goroutine。
//
// 幂等性：设备已停用时直接返回 nil。
// 仅标记会话为 disabled 并停止心跳，实际从 map 中移除由 cleanLoop 统一完成。
func (m *Manager) Disable(ctx context.Context, gatewaySn string) error {
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

// OnDeviceHeartbeat 设备心跳上行时调用，刷新 lastHeartbeat。
//
// 锁顺序：先释放 m.mu(RLock) 再获取 session.mu，避免交叉加锁。
// 状态不存在（设备未启用）时打印日志，不做推送前端。
// MaxDeadline 已过期时标记会话失效，实际从 map 中移除由 cleanLoop 统一完成。
func (m *Manager) OnDeviceHeartbeat(ctx context.Context, gatewaySn string) {
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

// GetNextSeq 获取并递增该设备的下一个 DRC 序号。
// 设备 DRC 未启用或已过期时返回 FailedPrecondition gRPC 错误。
// 并发安全：通过 session.IsAlive 确认存活状态，通过 atomic 递增序号。
func (m *Manager) GetNextSeq(gatewaySn string) (int, error) {
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

// GetStatus 查询设备 DRC 运行状态快照。
// 返回值为读取瞬间的快照，不保证后续一致性。
func (m *Manager) GetStatus(gatewaySn string) (enabled bool, startedAt, lastHb time.Time, nextSeq int, alive bool) {
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

// Close 停止 cleanLoop 和所有设备的心跳 goroutine。
// 所有心跳 ctx 都由 m.ctx 派生，取消父 ctx 会传播到全部子 ctx。
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		m.cancel()
	})
}

// startHeartbeatLocked 为设备启动心跳 goroutine。
// 调用方必须持有 m.mu（写锁）。
// 若该设备已有运行中的心跳 goroutine，会先停止旧的再启动新的，避免 goroutine 泄漏。
func (m *Manager) startHeartbeat(session *DeviceSession) {
	deadline := session.MaxDeadline
	sessionID := session.SessionID

	var heartbeatCtx context.Context
	var heartbeatCancel context.CancelFunc

	if !deadline.IsZero() {
		heartbeatCtx, heartbeatCancel = context.WithDeadline(m.ctx, deadline)
	} else {
		heartbeatCtx, heartbeatCancel = context.WithCancel(m.ctx)
	}
	session.heartbeatCancel = heartbeatCancel

	go m.heartbeatLoop(session.GatewaySn, sessionID, heartbeatCtx)
}

// cancelHeartbeat 停止设备的心跳 goroutine。
// 幂等：设备无运行中的 goroutine 时为空操作。
func (m *Manager) cancelHeartbeat(session *DeviceSession) {
	cancel := session.heartbeatCancel
	session.heartbeatCancel = nil

	if cancel != nil {
		cancel()
	}
}

// heartbeatLoop 设备心跳发送循环。
// 职责单一：定时检查设备存活状态，向设备下发心跳报文。
// 设备不再存活或会话不匹配时直接退出；退出时只清理自身 heartbeatCancel。
func (m *Manager) heartbeatLoop(gatewaySn, sessionID string, heartbeatCtx context.Context) {
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
			if m.djiClient == nil {
				continue
			}
			sendCtx, cancel := context.WithTimeout(heartbeatCtx, m.config.HeartbeatInterval)
			_, err := m.djiClient.SendDrcHeartBeat(sendCtx, gatewaySn, time.Now().UnixMilli())
			cancel()
			if err != nil {
				logx.Errorf("[drc-heartbeat] send failed: gateway_sn=%s err=%v", gatewaySn, err)
			} else {
				logx.Debugf("[drc-heartbeat] sent: gateway_sn=%s", gatewaySn)
			}
		}
	}
}

// isCurrentSessionAlive 检查指定设备的当前会话是否仍然存活。
// 用于 heartbeatLoop 中判断是否应继续发送心跳。
func (m *Manager) isCurrentSessionAlive(gatewaySn, sessionID string) bool {
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

// expireSession 在 MaxDeadline 到达时由 heartbeatLoop 调用，标记会话失效。
// 通过 sessionID 比对防止误清新会话。实际从 map 中移除由 cleanLoop 统一完成。
func (m *Manager) expireSession(gatewaySn, sessionID string) {
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

// cleanLoop 定期扫描所有设备状态，清理过期会话。
// 通过 time.Since(lastHeartbeat) > HeartbeatTimeout 判断设备离线。
// 扫描间隔自适应：clamp(HeartbeatTimeout/2, 5s, 15s)。
func (m *Manager) cleanLoop() {
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

// cleanupExpiredStates 清理所有过期的设备状态（唯一执行 delete(m.session) 的方法）。
//
// 先在锁内收集过期 session 并从 map 移除，再在锁外停止心跳和通知。
// Disable/OnDeviceHeartbeat/expireSession 仅标记 Enabled=false，由本方法统一回收。
func (m *Manager) cleanupExpiredStates() int {
	expired := make(map[string]*DeviceSession) // gatewaySn -> session

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

// loadOrInitSession 获取设备的 DeviceSession，优先从 states 读取，不存在时创建并注册到 states。
//
// 调用方必须持有 m.mu（写锁）。
func (m *Manager) loadOrInitSession(gatewaySn string) *DeviceSession {
	if session, ok := m.session[gatewaySn]; ok {
		return session
	}
	session := &DeviceSession{GatewaySn: gatewaySn}
	m.session[gatewaySn] = session
	return session
}

// fireSessionExpired 触发会话过期回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionExpired(gatewaySn, sessionID, reason string) {
	if m.onSessionExpired != nil {
		threading.GoSafe(func() {
			m.onSessionExpired(gatewaySn, sessionID, reason)
		})
	}
}

// fireSessionEnabled 触发会话启用回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionEnabled(gatewaySn, sessionID string) {
	if m.onSessionEnabled != nil {
		threading.GoSafe(func() {
			m.onSessionEnabled(gatewaySn, sessionID)
		})
	}
}

// fireSessionDisabled 触发会话停用回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionDisabled(gatewaySn, sessionID string) {
	if m.onSessionDisabled != nil {
		threading.GoSafe(func() {
			m.onSessionDisabled(gatewaySn, sessionID)
		})
	}
}
