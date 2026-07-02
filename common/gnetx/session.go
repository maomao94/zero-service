package gnetx

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/antsx"
)

// Session 是 gnetx 的每连接上下文，封装 gnet.Conn 并提供业务层 API。
// 用户 handler 收到 *Session，不直接接触底层 gnet.Conn。
//
// 生命周期：OnOpen 时由 Server/Client 创建 → 业务可调 Register 设 alias →
// OnClose 或 Close 时从 SessionManager 移除并通知 listener。
//
// 并发安全：Attribute/lastActive/Close 等方法可跨 goroutine 调用；
// Send/Notify/Request 内部用 AsyncWrite（off-loop 安全），可在业务 goroutine 调用。
// 禁止在 event-loop handler 中调用 Request（会阻塞 loop）。
type Session struct {
	id       string // 框架分配（远端地址派生）
	alias    string // opt-in 业务 id（设备号等），Register 时设置
	conn     gnet.Conn
	codec    Codec           // 编解码器，Send/Request 编码用（不依赖 mgr）
	mgr      *SessionManager // 会话管理器；client 单连接模型下为 nil
	isClient bool

	created    time.Time
	lastActive atomic.Int64 // unix nano

	attrs sync.Map

	// pool 是 opt-in 请求-响应的 ReplyPool，懒初始化（首次 Request 时建）。
	// 用 atomic.Pointer 保证 initPool（off-loop）写与 resolveResponse（on-loop）读之间无数据竞争。
	pool     atomic.Pointer[antsx.ReplyPool[any]]
	poolOnce sync.Once

	closeOnce sync.Once
	closed    atomic.Bool
}

// newSession 创建一个 Session 并记录创建时间与初始 lastActive。
// codec 用于 Send/Request 编码；mgr 可为 nil（client 单连接模型不需要管理器）。
func newSession(id string, conn gnet.Conn, codec Codec, mgr *SessionManager, isClient bool) *Session {
	now := time.Now()
	s := &Session{
		id:       id,
		conn:     conn,
		codec:    codec,
		mgr:      mgr,
		isClient: isClient,
		created:  now,
	}
	s.lastActive.Store(now.UnixNano())
	return s
}

// ID 返回框架分配的会话 id（通常由远端地址派生）。
func (s *Session) ID() string { return s.id }

// Alias 返回业务侧注册的 alias（设备号等）；未注册返回空串。
func (s *Session) Alias() string { return s.alias }

// IsClient 返回 true 表示这是 client 端拨号产生的 Session。
func (s *Session) IsClient() bool { return s.isClient }

// CreatedAt 返回会话创建时间。
func (s *Session) CreatedAt() time.Time { return s.created }

// LastActiveAt 返回最近一次收到数据的时间。
func (s *Session) LastActiveAt() time.Time {
	return time.Unix(0, s.lastActive.Load())
}

// RemoteAddr 返回远端地址。
func (s *Session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// LocalAddr 返回本地地址。
func (s *Session) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// SetAttribute 在会话上存储一个业务属性（typed key 推荐用自定义 key 类型）。
// 并发安全。
func (s *Session) SetAttribute(key, val any) {
	s.attrs.Store(key, val)
}

// Attribute 读取会话上的业务属性，不存在返回 nil。
func (s *Session) Attribute(key any) any {
	v, _ := s.attrs.Load(key)
	return v
}

// DeleteAttribute 删除会话上的业务属性。
func (s *Session) DeleteAttribute(key any) {
	s.attrs.Delete(key)
}

// Register 把会话以 alias（业务 id，如设备号）登记到 SessionManager。
// 登记后可通过 SessionManager.Get(alias) 查到。若 alias 已被占用，旧会话会被踢掉。
// 仅 opt-in：无设备身份概念的协议可不调。
func (s *Session) Register(alias string) {
	s.alias = alias
	if s.mgr != nil {
		s.mgr.registerAlias(s, alias)
	}
}

// Send 把消息编码后通过 AsyncWrite 发送（fire-and-forget）。
// off-loop 安全，可在业务 goroutine 调用。会话已关闭返回 ErrSessionClosed。
func (s *Session) Send(msg any) error {
	if s.closed.Load() {
		return ErrSessionClosed
	}
	payload, err := s.codec.Encode(msg, s)
	if err != nil {
		return err
	}
	return s.conn.AsyncWrite(payload, nil)
}

// Notify 是 Send 的语义别名，对齐 antsx 的 fire-and-forget 命名。
func (s *Session) Notify(_ context.Context, msg any) error {
	return s.Send(msg)
}

// Request 发送请求并等待匹配 tid 的回包（opt-in 请求-响应）。
// msg 必须实现 Correlatable；回包需实现 Response 且 ResponseTID 与 msg.TID 一致。
// ttl 控制等待时长；ctx 控制取消。
//
// 关联引擎为每 Session 一个 antsx.ReplyPool，生命周期绑 Session，断连自动 Reject 在途。
//
// 线程约束（重要）：Request 会阻塞等待回包，**只能在 off-loop 调用**——
// 即业务 goroutine（通过 SessionManager.Get 拿到的 Session）或 AsyncHandler 内。
// 严禁在同步 handler（on-loop）里调用：会阻塞 event-loop，直到 ctx/ttl 超时才恢复，
// 期间同 loop 上所有连接停摆。gnet 不暴露 on-loop 判定，框架无法拦截，由调用方保证。
func (s *Session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
	if s.closed.Load() {
		return nil, ErrSessionClosed
	}
	pool := s.ensurePool()
	if pool == nil {
		return nil, ErrSessionClosed
	}
	tid := msg.TID()
	return antsx.RequestReply[any](ctx, pool, tid, func() error { return s.Send(msg) }, ttl)
}

// ensurePool 懒初始化并返回 ReplyPool。仅首次 Request 时创建，纯推送协议不创建。
// 用 atomic.Pointer 存储，与 resolveResponse 的并发读无竞争。
// 若创建后发现 Session 已关闭，立即 Close 新池并返回 nil，避免 TimingWheel 泄漏。
func (s *Session) ensurePool() *antsx.ReplyPool[any] {
	s.poolOnce.Do(func() {
		p := antsx.NewReplyPool[any](
			antsx.WithName("gnetx-"+s.id),
			antsx.WithDefaultTTL(30*time.Second),
		)
		s.pool.Store(p)
		if s.closed.Load() {
			// 与 Close 竞争：Close 可能已跳过 pool。这里补偿关闭，防止泄漏。
			p.Close()
		}
	})
	return s.pool.Load()
}

// resolveResponse 把入站回包匹配到在途请求并完成对应 Promise。
// 返回 true 表示命中在途请求（已 Resolve）；false 表示无在途匹配（pool 未建或 tid 无对应）。
// 在 OnTraffic（on-loop）中调用。用 atomic 读取 pool，无数据竞争。
func (s *Session) resolveResponse(tid string, resp any) bool {
	pool := s.pool.Load()
	if pool == nil {
		return false
	}
	return pool.Resolve(tid, resp)
}

// touch 更新最近活跃时间。在 OnTraffic 中调用。
func (s *Session) touch() {
	s.lastActive.Store(time.Now().UnixNano())
}

// Close 幂等关闭会话：从 SessionManager 移除、关闭 ReplyPool、关闭底层连接。
// 触发 listener.OnDestroyed。off-loop 安全（gnet.Conn.Close 跨 goroutine 安全）。
func (s *Session) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.mgr != nil {
			s.mgr.remove(s)
		}
		if pool := s.pool.Load(); pool != nil {
			pool.Close()
		}
		closeErr = s.conn.Close()
	})
	return closeErr
}

// isClosed 返回会话是否已关闭。
func (s *Session) isClosed() bool {
	return s.closed.Load()
}

// SessionManager 管理所有活跃 Session，支持按框架 id 和业务 alias 查找。
// 并发安全。Server 持有一个实例；client 单连接模型不使用。
type SessionManager struct {
	mu       sync.RWMutex
	byID     map[string]*Session
	byAlias  map[string]*Session
	listener SessionListener
}

// NewSessionManager 创建会话管理器。listener 可传 nil（默认 noop）。
func NewSessionManager(listener SessionListener) *SessionManager {
	if listener == nil {
		listener = noopSessionListener{}
	}
	return &SessionManager{
		byID:     make(map[string]*Session),
		byAlias:  make(map[string]*Session),
		listener: listener,
	}
}

// Get 按 id 或 alias 查找会话。先查 alias 再查 byID。
func (m *SessionManager) Get(idOrAlias string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.byAlias[idOrAlias]; ok {
		return s
	}
	return m.byID[idOrAlias]
}

// All 返回当前所有活跃会话的快照。用于广播或空闲扫描。
func (m *SessionManager) All() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.byID))
	for _, s := range m.byID {
		out = append(out, s)
	}
	return out
}

// Count 返回当前活跃会话数。
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byID)
}

// add 把会话加入管理器（按框架 id），并触发 listener.OnCreated。
// 由 Server/Client 在 OnOpen 时调用。
func (m *SessionManager) add(s *Session) {
	m.mu.Lock()
	m.byID[s.id] = s
	m.mu.Unlock()
	m.listener.OnCreated(s)
}

// registerAlias 把会话按业务 alias 登记，并触发 listener.OnRegistered。
// 若 alias 已被占用，旧会话被踢掉（Close）。
func (m *SessionManager) registerAlias(s *Session, alias string) {
	m.mu.Lock()
	if old, ok := m.byAlias[alias]; ok && old != s {
		m.mu.Unlock()
		// 踢掉旧会话：在不持有锁时 Close，避免死锁。
		_ = old.Close()
		m.mu.Lock()
	}
	m.byAlias[alias] = s
	m.mu.Unlock()
	m.listener.OnRegistered(s)
}

// remove 把会话从管理器移除（按 id 和 alias），并触发 listener.OnDestroyed。
func (m *SessionManager) remove(s *Session) {
	m.mu.Lock()
	delete(m.byID, s.id)
	if s.alias != "" {
		if cur, ok := m.byAlias[s.alias]; ok && cur == s {
			delete(m.byAlias, s.alias)
		}
	}
	m.mu.Unlock()
	m.listener.OnDestroyed(s)
}

// SessionListener 监听会话生命周期事件。所有方法默认空实现，用户按需 override。
// 嵌入 noopSessionListener 即可获得全部默认空实现。
type SessionListener interface {
	OnCreated(s *Session)    // OnOpen 时
	OnRegistered(s *Session) // Session.Register 时
	OnDestroyed(s *Session)  // OnClose/Close 时
}

// noopSessionListener 提供 SessionListener 的全部空实现。
type noopSessionListener struct{}

func (noopSessionListener) OnCreated(*Session)    {}
func (noopSessionListener) OnRegistered(*Session) {}
func (noopSessionListener) OnDestroyed(*Session)  {}

// logSessionClosed 记录会话关闭日志，统一格式。
func logSessionClosed(s *Session, cause string) {
	logx.Infof("[gnetx] session closed id=%s alias=%s remote=%s cause=%s",
		s.id, s.alias, s.RemoteAddr(), cause)
}
