package drc

import (
	"context"
	"sync"
	"time"
	"zero-service/common/tool"

	"zero-service/app/djicloud/internal/config"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
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
//   - m.mu 保护 cache 读写与心跳 goroutine 启停的原子性，防止 Enable/Disable/OnDeviceHeartbeat 交叉操作
//   - state.mu 保护单个设备的字段级并发读写（Enabled、seq、时间戳等）
//   - sync.Map(cancels) 用于心跳取消函数的无锁查找，仅在 m.mu 持有时做 Store/Delete
//
// 设备心跳超时通过 collection.Cache 的 TTL 自动管理：
//   - 收到设备心跳上行时调用 cache.Set 刷新 TTL
//   - TTL（HeartbeatTimeout）过期后条目自动驱逐，等价于设备离线
//
// 心跳下发 goroutine 使用 context.Context 控制生命周期：
//   - Enable 时构造子 ctx（含最大超时 deadline 或普通 cancel）
//   - 心跳循环通过 ctx.Done() 监听退出信号
//   - 全局 cleanLoop 定期扫描孤儿 goroutine（缓存已过期但 goroutine 仍存活）
type Manager struct {
	mu sync.Mutex
	// cache 设备 DRC 状态缓存，key=gatewaySn, value=*State, TTL=HeartbeatTimeout。
	// 条目存在且非过期即表示设备 DRC 存活（收到过心跳且未超时）。
	cache *collection.Cache
	// cancels 跟踪正在运行的心跳 goroutine 的取消函数。
	// key: gatewaySn(string), value: context.CancelFunc。
	// 仅在持有 m.mu 时做 Store/Delete，cleanLoop 中的 Range+cancel 为安全降级操作。
	cancels   sync.Map
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
	ctx    context.Context
	cancel context.CancelFunc
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
	cache, err := collection.NewCache(cfg.HeartbeatTimeout, collection.WithName("drc-cache"))
	if err != nil {
		logx.Errorf("[drc-manager] failed to create cache: %v", err)
		cache, _ = collection.NewCache(300 * time.Second)
	}
	ctx, mCancel := context.WithCancel(context.Background())
	m := &Manager{
		cache:     cache,
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

	state := m.loadOrInitState(gatewaySn)
	state.mu.Lock()
	if state.Enabled && !state.IsExpired(time.Now(), m.config.HeartbeatTimeout) {
		logx.WithContext(ctx).Debugf("[drc-manager] enable skipped (already alive): gateway_sn=%s session_id=%s", gatewaySn, state.SessionID)
		state.mu.Unlock()
		return nil
	}
	needStopOld := state.Enabled
	state.mu.Unlock()

	if needStopOld {
		logx.WithContext(ctx).Debugf("[drc-manager] stopping stale heartbeat before re-enable: gateway_sn=%s", gatewaySn)
		m.stopHeartbeatLocked(gatewaySn)
	}

	state.mu.Lock()
	state.Enabled = true
	state.StartedAt = time.Now()
	state.LastDeviceHeartbeat = state.StartedAt
	sessionId, _ := tool.SimpleUUID()
	state.SessionID = sessionId
	state.seq = 0
	if opt.maxTimeout > 0 {
		state.MaxDeadline = state.StartedAt.Add(opt.maxTimeout)
	} else {
		state.MaxDeadline = time.Time{}
	}
	sessionID := state.SessionID
	m.cache.Set(gatewaySn, state)
	state.mu.Unlock()

	m.startHeartbeatLocked(gatewaySn, sessionID)
	logx.WithContext(ctx).Infof("[drc-manager] enabled: gateway_sn=%s session_id=%s max_timeout=%v", gatewaySn, sessionID, opt.maxTimeout)
	m.fireSessionEnabled(gatewaySn, sessionID)
	return nil
}

// Disable 停用设备的 DRC 模式并停止心跳 goroutine。
//
// 幂等性：设备已停用时直接返回 nil。
func (m *Manager) Disable(ctx context.Context, gatewaySn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.loadOrInitState(gatewaySn)
	state.mu.Lock()
	if !state.Enabled {
		logx.WithContext(ctx).Debugf("[drc-manager] disable skipped (already disabled): gateway_sn=%s", gatewaySn)
		state.mu.Unlock()
		return nil
	}
	sessionID := state.SessionID
	state.Enabled = false
	state.mu.Unlock()

	m.cache.Del(gatewaySn)
	m.stopHeartbeatLocked(gatewaySn)
	logx.WithContext(ctx).Infof("[drc-manager] disabled: gateway_sn=%s session_id=%s", gatewaySn, sessionID)
	m.fireSessionDisabled(gatewaySn, sessionID)
	return nil
}

// OnDeviceHeartbeat 设备心跳上行时调用，刷新存活超时与 LastDeviceHeartbeat。
//
// 使用 m.mu 保护缓存读写一致性，避免与 Disable 并发导致缓存重建。
// 缓存不存在（设备未启用或 TTL 已过期）时打印日志，不做推送前端。
// MaxDeadline 已过期时主动清除会话。
func (m *Manager) OnDeviceHeartbeat(ctx context.Context, gatewaySn string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		logx.WithContext(ctx).Infof("[drc-manager] device heartbeat received but cache miss (not enabled or expired): %s", gatewaySn)
		return
	}

	state := val.(*State)
	state.mu.Lock()
	now := time.Now()
	if !state.MaxDeadline.IsZero() && now.After(state.MaxDeadline) {
		logx.WithContext(ctx).Infof("[drc-manager] device heartbeat rejected (max deadline exceeded): gateway_sn=%s session_id=%s", gatewaySn, state.SessionID)
		sessionID := state.SessionID
		state.Enabled = false
		state.mu.Unlock()
		m.cache.Del(gatewaySn)
		m.stopHeartbeatLocked(gatewaySn)
		m.fireSessionExpired(gatewaySn, sessionID, "max_deadline_exceeded")
		return
	}
	state.LastDeviceHeartbeat = now
	state.mu.Unlock()
	m.cache.Set(gatewaySn, state)
	logx.WithContext(ctx).Debugf("[drc-manager] device heartbeat refreshed: gateway_sn=%s", gatewaySn)
}

// GetNextSeq 获取并递增该设备的下一个 DRC 序号。
// 设备 DRC 未启用或缓存已过期时返回 FailedPrecondition gRPC 错误。
// 并发安全：通过 state.mu 保护序号的原子递增。
func (m *Manager) GetNextSeq(gatewaySn string) (int, error) {
	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		return 0, status.Errorf(codes.FailedPrecondition,
			"DRC mode not enabled for device=%s, please call DrcModeEnter first", gatewaySn)
	}

	state := val.(*State)
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.Enabled || state.IsExpired(time.Now(), m.config.HeartbeatTimeout) {
		return 0, status.Errorf(codes.FailedPrecondition,
			"DRC mode not enabled for device=%s, please call DrcModeEnter first", gatewaySn)
	}
	seq := state.seq
	state.seq++
	return seq, nil
}

// IsAlive 判断设备 DRC 是否存活：已启用且缓存未过期（心跳 TTL）且未超过 MaxDeadline。
// 并发安全：collection.Cache 本身线程安全，state.mu 保护字段读取。
func (m *Manager) IsAlive(gatewaySn string) bool {
	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		return false
	}
	state := val.(*State)
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.Enabled && !state.IsExpired(time.Now(), m.config.HeartbeatTimeout)
}

// GetStatus 查询设备 DRC 运行状态快照。
// 返回值为读取瞬间的快照，不保证后续一致性。
func (m *Manager) GetStatus(gatewaySn string) (enabled bool, startedAt, lastHb time.Time, nextSeq int, alive bool) {
	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		return false, time.Time{}, time.Time{}, 0, false
	}

	state := val.(*State)
	state.mu.Lock()
	defer state.mu.Unlock()

	enabled = state.Enabled && !state.IsExpired(time.Now(), m.config.HeartbeatTimeout)
	startedAt = state.StartedAt
	lastHb = state.LastDeviceHeartbeat
	nextSeq = state.seq
	alive = enabled
	return
}

// Close 停止所有设备的心跳 goroutine 和 cleanLoop。
// 先取消全局 ctx（通过派生 ctx 传播到所有心跳 goroutine 和 cleanLoop），
// 再逐个调用心跳取消函数确保立即退出。
func (m *Manager) Close() {
	m.cancel()
	var cancels []context.CancelFunc
	m.cancels.Range(func(key, value any) bool {
		cancels = append(cancels, value.(context.CancelFunc))
		return true
	})
	for _, c := range cancels {
		c()
	}
}

// startHeartbeatLocked 为设备启动心跳 goroutine。
// 调用方必须持有 m.mu。
// 若该设备已有运行中的心跳 goroutine，会先停止旧的再启动新的，避免 goroutine 泄漏。
func (m *Manager) startHeartbeatLocked(gatewaySn, sessionID string) {
	m.stopHeartbeatLocked(gatewaySn)

	var heartbeatCtx context.Context
	var heartbeatCancel context.CancelFunc

	if val, ok := m.cache.Get(gatewaySn); ok {
		state := val.(*State)
		state.mu.Lock()
		deadline := state.MaxDeadline
		state.mu.Unlock()
		if !deadline.IsZero() {
			heartbeatCtx, heartbeatCancel = context.WithDeadline(m.ctx, deadline)
		} else {
			heartbeatCtx, heartbeatCancel = context.WithCancel(m.ctx)
		}
	} else {
		heartbeatCtx, heartbeatCancel = context.WithCancel(m.ctx)
	}

	m.cancels.Store(gatewaySn, heartbeatCancel)
	go m.heartbeatLoop(gatewaySn, sessionID, heartbeatCtx, heartbeatCancel)
}

// stopHeartbeatLocked 停止设备的心跳 goroutine。
// 调用方必须持有 m.mu。
// 幂等：设备无运行中的 goroutine 时为空操作。
func (m *Manager) stopHeartbeatLocked(gatewaySn string) {
	val, ok := m.cancels.Load(gatewaySn)
	if !ok {
		return
	}
	cancel := val.(context.CancelFunc)
	m.cancels.Delete(gatewaySn)
	cancel()
}

// heartbeatLoop 设备心跳发送循环。
// 职责单一：定时从缓存获取 DRC 状态，向设备下发心跳报文。
// 缓存 miss（TTL 过期）或会话不匹配时直接退出，由 cleanLoop 兜底清理孤儿 goroutine。
func (m *Manager) heartbeatLoop(gatewaySn, sessionID string, heartbeatCtx context.Context, heartbeatCancel context.CancelFunc) {
	ticker := time.NewTicker(m.config.HeartbeatInterval)
	defer ticker.Stop()
	defer heartbeatCancel()
	defer m.cancels.Delete(gatewaySn)

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
				logx.Infof("[drc-heartbeat] cache miss or stale session, stop: gateway_sn=%s session_id=%s", gatewaySn, sessionID)
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
	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		return false
	}
	state := val.(*State)
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.Enabled && state.SessionID == sessionID && !state.IsExpired(time.Now(), m.config.HeartbeatTimeout)
}

// expireSession 在 MaxDeadline 到达时由 heartbeatLoop 调用，清除过期会话。
// 通过 sessionID 比对防止误清新会话。
func (m *Manager) expireSession(gatewaySn, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.cache.Get(gatewaySn)
	if !ok {
		m.stopHeartbeatLocked(gatewaySn)
		return
	}
	state := val.(*State)
	state.mu.Lock()
	if state.SessionID != sessionID {
		state.mu.Unlock()
		return
	}
	state.Enabled = false
	state.mu.Unlock()
	m.cache.Del(gatewaySn)
	m.stopHeartbeatLocked(gatewaySn)
	m.fireSessionExpired(gatewaySn, sessionID, "max_deadline_exceeded")
}

// cleanLoop 定期扫描心跳 goroutine 与缓存的一致性，清理孤儿 goroutine。
// 场景：缓存因 TTL 过期被自动驱逐，但 heartbeatLoop 因 tick 间隔恰好越过检查。
func (m *Manager) cleanLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			logx.Debugf("[drc-clean] scanning for orphan heartbeat goroutines")
			m.cancels.Range(func(key, value any) bool {
				gatewaySn := key.(string)
				if _, ok := m.cache.Get(gatewaySn); !ok {
					logx.Infof("[drc-clean] orphan heartbeat goroutine detected, cancel: %s", gatewaySn)
					cancel := value.(context.CancelFunc)
					cancel()
					m.fireSessionExpired(gatewaySn, "", "heartbeat_timeout")
				}
				return true
			})
		}
	}
}

// loadOrInitState 获取设备的 State，优先从 cache 读取，不存在时返回临时零值对象。
//
// 调用方必须持有 m.mu。
// 注意：返回的对象仅当在 Enable / OnDeviceHeartbeat 中显式调用 cache.Set 后才进入缓存。
// 对于未启用 DRC 的设备，返回的 State.Enabled 为 false，不会被缓存。
func (m *Manager) loadOrInitState(gatewaySn string) *State {
	if val, ok := m.cache.Get(gatewaySn); ok {
		return val.(*State)
	}
	return &State{GatewaySn: gatewaySn}
}

// fireSessionExpired 触发会话过期回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionExpired(gatewaySn, sessionID, reason string) {
	if m.onSessionExpired != nil {
		go m.onSessionExpired(gatewaySn, sessionID, reason)
	}
}

// fireSessionEnabled 触发会话启用回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionEnabled(gatewaySn, sessionID string) {
	if m.onSessionEnabled != nil {
		go m.onSessionEnabled(gatewaySn, sessionID)
	}
}

// fireSessionDisabled 触发会话停用回调（如果已注册）。
// 在 goroutine 中异步执行，避免阻塞调用方持有的锁。
func (m *Manager) fireSessionDisabled(gatewaySn, sessionID string) {
	if m.onSessionDisabled != nil {
		go m.onSessionDisabled(gatewaySn, sessionID)
	}
}
