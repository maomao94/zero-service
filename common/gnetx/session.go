package gnetx

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/antsx"
)

func newSessionID() string {
	return uuid.New().String()
}

// Conn is the shared connection interface used by Handler for reuse across Server and Client.
// Both ServerConn and ClientConn embed Conn.
type Conn interface {
	SessionID() string
	NextSendSeq() uint64
	Write(ctx context.Context, msg any) error
	WriteAsync(ctx context.Context, msg any) error
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	CreatedAt() time.Time
	LastActiveAt() time.Time
	SetAttribute(key, val any)
	Attribute(key any) any
	DeleteAttribute(key any)
	Close() error
}

// session is the concrete per-connection context implementing Conn, ServerConn, and ClientConn.
type session struct {
	sessionID  string
	clientID   string
	gc         gnet.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
	codec      Codec
	mgr        *SessionManager
	created    time.Time
	lastActive atomic.Int64
	sendSeq    atomic.Uint64

	attrs sync.Map

	replyPool *antsx.ReplyPool[any]

	stateMu   sync.RWMutex
	closeOnce sync.Once
	closed    bool
}

func newSession(sessionID string, gc gnet.Conn, codec Codec, mgr *SessionManager, replyPool *antsx.ReplyPool[any], sequenceStart ...uint64) *session {
	now := time.Now()
	s := &session{
		sessionID:  sessionID,
		gc:         gc,
		localAddr:  snapshotAddr(gc.LocalAddr()),
		remoteAddr: snapshotAddr(gc.RemoteAddr()),
		codec:      codec,
		mgr:        mgr,
		replyPool:  replyPool,
		created:    now,
	}
	s.lastActive.Store(now.UnixNano())
	if len(sequenceStart) > 0 {
		s.sendSeq.Store(sequenceStart[0])
	}
	return s
}

func (s *session) SessionID() string         { return s.sessionID }
func (s *session) NextSendSeq() uint64       { return s.sendSeq.Add(1) - 1 }
func (s *session) CreatedAt() time.Time      { return s.created }
func (s *session) LastActiveAt() time.Time   { return time.Unix(0, s.lastActive.Load()) }
func (s *session) RemoteAddr() net.Addr      { return s.remoteAddr }
func (s *session) LocalAddr() net.Addr       { return s.localAddr }
func (s *session) SetAttribute(key, val any) { s.attrs.Store(key, val) }
func (s *session) DeleteAttribute(key any)   { s.attrs.Delete(key) }

func (s *session) Attribute(key any) any {
	v, _ := s.attrs.Load(key)
	return v
}

func (s *session) encode(ctx context.Context, msg any) ([]byte, error) {
	if s.isClosed() {
		return nil, ErrSessionClosed
	}
	payload, err := s.codec.Encode(ctx, msg, s)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *session) Write(ctx context.Context, msg any) error {
	payload, err := s.encode(ctx, msg)
	if err != nil {
		return err
	}
	_, err = s.gc.Write(payload)
	return err
}

func (s *session) WriteAsync(ctx context.Context, msg any) error {
	payload, err := s.encode(ctx, msg)
	if err != nil {
		return err
	}
	return s.gc.AsyncWrite(payload, nil)
}

func (s *session) ClientID() string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.clientID
}

func (s *session) bindClientID(clientID string) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.closed {
		return ErrSessionClosed
	}
	s.clientID = clientID
	return nil
}

func (s *session) touch()         { s.lastActive.Store(time.Now().UnixNano()) }
func (s *session) isClosed() bool { s.stateMu.RLock(); defer s.stateMu.RUnlock(); return s.closed }
func (s *session) markClosed()    { s.stateMu.Lock(); s.closed = true; s.stateMu.Unlock() }

func (s *session) BindClientID(clientID string) error {
	if clientID == "" {
		return ErrInvalidClientID
	}
	if s.mgr == nil {
		return s.bindClientID(clientID)
	}
	return s.mgr.bindClientID(s, clientID)
}

func (s *session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
	if s.isClosed() {
		return nil, ErrSessionClosed
	}
	if s.replyPool == nil {
		return nil, ErrSessionClosed
	}
	ctx = injectSessionLogFields(ctx, s)
	compositeTID := s.sessionID + "|" + msg.TID()
	return antsx.RequestReply[any](ctx, s.replyPool, compositeTID, func() error { return s.WriteAsync(ctx, msg) }, ttl)
}

func (s *session) resolveResponse(tid string, resp any) bool {
	if tid == "" {
		return false
	}
	if s.replyPool == nil {
		return false
	}
	compositeTID := s.sessionID + "|" + tid
	return s.replyPool.Resolve(compositeTID, resp)
}

func (s *session) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.closeState()
		closeErr = s.gc.Close()
	})
	return closeErr
}

func (s *session) closeFromEventLoop() {
	s.closeOnce.Do(s.closeState)
}

func (s *session) closeState() {
	if s.mgr != nil {
		s.mgr.remove(s)
	} else {
		s.markClosed()
	}
}

type immutableAddr struct {
	network string
	address string
}

func (a immutableAddr) Network() string { return a.network }
func (a immutableAddr) String() string  { return a.address }

func snapshotAddr(addr net.Addr) net.Addr {
	if addr == nil {
		return nil
	}
	return immutableAddr{network: addr.Network(), address: addr.String()}
}

// SessionManager manages active sessions. Used by Server (multi-session). Thread-safe.
type SessionManager struct {
	mu          sync.RWMutex
	bySessionID map[string]*session
	byClientID  map[string]*session
	listener    SessionListener
}

func NewSessionManager(listener SessionListener) *SessionManager {
	if listener == nil {
		listener = noopSessionListener{}
	}
	return &SessionManager{
		bySessionID: make(map[string]*session),
		byClientID:  make(map[string]*session),
		listener:    listener,
	}
}

func (m *SessionManager) GetBySessionID(sessionID string) Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.bySessionID[sessionID]
	if !ok || s.isClosed() {
		return nil
	}
	return s
}

func (m *SessionManager) GetByClientID(clientID string) Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.byClientID[clientID]
	if !ok || s.isClosed() {
		return nil
	}
	return s
}

func (m *SessionManager) All() []Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Conn, 0, len(m.bySessionID))
	for _, s := range m.bySessionID {
		if !s.isClosed() {
			out = append(out, s)
		}
	}
	return out
}

func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.bySessionID {
		if !s.isClosed() {
			count++
		}
	}
	return count
}

func (m *SessionManager) add(s *session) {
	m.mu.Lock()
	if s.isClosed() {
		m.mu.Unlock()
		return
	}
	m.bySessionID[s.sessionID] = s
	m.mu.Unlock()
	m.listener.OnCreated(s)
}

func (m *SessionManager) bindClientID(s *session, clientID string) error {
	m.mu.Lock()
	if current := m.bySessionID[s.sessionID]; current != s {
		m.mu.Unlock()
		return ErrSessionClosed
	}
	previousID := s.ClientID()
	if err := s.bindClientID(clientID); err != nil {
		m.mu.Unlock()
		return err
	}
	if previousID != "" && previousID != clientID {
		if current := m.byClientID[previousID]; current == s {
			delete(m.byClientID, previousID)
		}
	}
	old := m.byClientID[clientID]
	if old != nil && old != s {
		// Mark the displaced session while holding the same lock used by
		// BindClientID. Its delayed Close must not be able to rebind later.
		old.markClosed()
	}
	m.byClientID[clientID] = s
	m.mu.Unlock()

	if old != nil && old != s {
		_ = old.Close()
	}
	m.listener.OnRegistered(s)
	return nil
}

func (m *SessionManager) remove(s *session) {
	m.mu.Lock()
	// Session owns the lifecycle state; manager lock serializes it with BindClientID.
	s.markClosed()
	delete(m.bySessionID, s.sessionID)
	if clientID := s.ClientID(); clientID != "" {
		if current := m.byClientID[clientID]; current == s {
			delete(m.byClientID, clientID)
		}
	}
	m.mu.Unlock()
	m.listener.OnDestroyed(s)
}

// injectSessionLogFields 将 session 级信息注入 context，使 handler 内
// logx.WithContext(ctx) 自动携带 sessionID、local、remote、clientID。
func injectSessionLogFields(ctx context.Context, s *session) context.Context {
	ctx = logx.ContextWithFields(ctx,
		logx.Field("sessionID", s.sessionID),
		logx.Field("local", s.LocalAddr().String()),
		logx.Field("remote", s.RemoteAddr().String()),
	)
	if clientID := s.ClientID(); clientID != "" {
		ctx = logx.ContextWithFields(ctx, logx.Field("clientID", clientID))
	}
	return ctx
}
