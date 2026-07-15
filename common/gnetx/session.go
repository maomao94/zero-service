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
	ID() string
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
	id         string
	alias      string
	gc         gnet.Conn
	codec      Codec
	mgr        *SessionManager
	created    time.Time
	lastActive atomic.Int64
	sendSeq    atomic.Uint64

	attrs sync.Map

	replyPool *antsx.ReplyPool[any]

	closeOnce sync.Once
	closed    atomic.Bool
	closeFunc func() // called on Close by Dialer to stop gnet.Client
}

func newSession(id string, gc gnet.Conn, codec Codec, mgr *SessionManager, replyPool *antsx.ReplyPool[any], sequenceStart ...uint64) *session {
	now := time.Now()
	s := &session{
		id:        id,
		gc:        gc,
		codec:     codec,
		mgr:       mgr,
		replyPool: replyPool,
		created:   now,
	}
	s.lastActive.Store(now.UnixNano())
	if len(sequenceStart) > 0 {
		s.sendSeq.Store(sequenceStart[0])
	}
	return s
}

func (s *session) ID() string                { return s.id }
func (s *session) NextSendSeq() uint64       { return s.sendSeq.Add(1) - 1 }
func (s *session) CreatedAt() time.Time      { return s.created }
func (s *session) LastActiveAt() time.Time   { return time.Unix(0, s.lastActive.Load()) }
func (s *session) RemoteAddr() net.Addr      { return s.gc.RemoteAddr() }
func (s *session) LocalAddr() net.Addr       { return s.gc.LocalAddr() }
func (s *session) SetAttribute(key, val any) { s.attrs.Store(key, val) }
func (s *session) DeleteAttribute(key any)   { s.attrs.Delete(key) }

func (s *session) Attribute(key any) any {
	v, _ := s.attrs.Load(key)
	return v
}

func (s *session) encode(ctx context.Context, msg any) ([]byte, error) {
	if s.closed.Load() {
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

func (s *session) Alias() string  { return s.alias }
func (s *session) touch()         { s.lastActive.Store(time.Now().UnixNano()) }
func (s *session) isClosed() bool { return s.closed.Load() }

func (s *session) Register(alias string) {
	s.alias = alias
	if s.mgr != nil {
		s.mgr.registerAlias(s, alias)
	}
}

func (s *session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
	if s.closed.Load() {
		return nil, ErrSessionClosed
	}
	if s.replyPool == nil {
		return nil, ErrSessionClosed
	}
	ctx = injectSessionLogFields(ctx, s)
	compositeTID := s.id + "|" + msg.TID()
	return antsx.RequestReply[any](ctx, s.replyPool, compositeTID, func() error { return s.WriteAsync(ctx, msg) }, ttl)
}

func (s *session) resolveResponse(tid string, resp any) bool {
	if tid == "" {
		return false
	}
	if s.replyPool == nil {
		return false
	}
	compositeTID := s.id + "|" + tid
	return s.replyPool.Resolve(compositeTID, resp)
}

func (s *session) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.mgr != nil {
			s.mgr.remove(s)
		}
		closeErr = s.gc.Close()
		if s.closeFunc != nil {
			s.closeFunc()
		}
	})
	return closeErr
}

// SessionManager manages active sessions. Used by Server (multi-session). Thread-safe.
type SessionManager struct {
	mu       sync.RWMutex
	byID     map[string]*session
	byAlias  map[string]*session
	listener SessionListener
}

func NewSessionManager(listener SessionListener) *SessionManager {
	if listener == nil {
		listener = noopSessionListener{}
	}
	return &SessionManager{
		byID:     make(map[string]*session),
		byAlias:  make(map[string]*session),
		listener: listener,
	}
}

func (m *SessionManager) Get(idOrAlias string) Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.byAlias[idOrAlias]; ok {
		return s
	}
	if s, ok := m.byID[idOrAlias]; ok {
		return s
	}
	return nil
}

func (m *SessionManager) All() []Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Conn, 0, len(m.byID))
	for _, s := range m.byID {
		out = append(out, s)
	}
	return out
}

func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byID)
}

func (m *SessionManager) add(s *session) {
	m.mu.Lock()
	m.byID[s.id] = s
	m.mu.Unlock()
	m.listener.OnCreated(s)
}

func (m *SessionManager) registerAlias(s *session, alias string) {
	m.mu.Lock()
	if old, ok := m.byAlias[alias]; ok && old != s {
		m.mu.Unlock()
		_ = old.Close()
		m.mu.Lock()
	}
	m.byAlias[alias] = s
	m.mu.Unlock()
	m.listener.OnRegistered(s)
}

func (m *SessionManager) remove(s *session) {
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

// injectSessionLogFields 将 session 级信息注入 context，使 handler 内
// logx.WithContext(ctx) 自动携带 session_id、local、remote、alias。
func injectSessionLogFields(ctx context.Context, s *session) context.Context {
	ctx = logx.ContextWithFields(ctx,
		logx.Field("session_id", s.id),
		logx.Field("local", s.LocalAddr().String()),
		logx.Field("remote", s.RemoteAddr().String()),
	)
	if alias := s.Alias(); alias != "" {
		ctx = logx.ContextWithFields(ctx, logx.Field("alias", alias))
	}
	return ctx
}
