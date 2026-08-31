package livekitx

import (
	"context"
	"sync"
	"time"
)

// ConnectionStatus 描述实时连接在业务 Store 中的可序列化状态。
type ConnectionStatus string

const (
	// ConnectionStatusConnecting 表示连接正在建立中。
	ConnectionStatusConnecting ConnectionStatus = "connecting"
	// ConnectionStatusConnected 表示连接已建立。
	ConnectionStatusConnected ConnectionStatus = "connected"
	// ConnectionStatusClosed 表示连接已关闭。
	ConnectionStatusClosed ConnectionStatus = "closed"
)

// ConnectionState 只保存连接节点和会话标识，不保存不可序列化的 SDK Room 指针。
type ConnectionState struct {
	NodeID, RoomName, Identity, SessionID string
	Status                                ConnectionStatus
	UpdatedAt, ExpiresAt                  time.Time
}

// Store 保存活动实时连接的业务可序列化索引；实现可接入 Redis/DB。
type Store interface {
	Put(context.Context, ConnectionState) error
	Get(context.Context, string) (ConnectionState, bool, error)
	Delete(context.Context, string) error
	Close() error
}

type memoryStore struct {
	mu     sync.RWMutex
	closed bool
	states map[string]ConnectionState
}

// NewMemoryStore 创建单机内存 Store，适合单测和单节点部署。
func NewMemoryStore() Store { return &memoryStore{states: make(map[string]ConnectionState)} }

func (s *memoryStore) Put(ctx context.Context, state ConnectionState) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if state.SessionID == "" {
		return ErrInvalidConfig
	}
	s.states[state.SessionID] = state
	return nil
}

func (s *memoryStore) Get(ctx context.Context, id string) (ConnectionState, bool, error) {
	if err := contextErr(ctx); err != nil {
		return ConnectionState{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ConnectionState{}, false, ErrClosed
	}
	state, ok := s.states[id]
	if ok && !state.ExpiresAt.IsZero() && !state.ExpiresAt.After(time.Now()) {
		delete(s.states, id)
		ok = false
	}
	return state, ok, nil
}

func (s *memoryStore) Delete(ctx context.Context, id string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	delete(s.states, id)
	return nil
}

func (s *memoryStore) Close() error {
	s.mu.Lock()
	s.closed = true
	s.states = nil
	s.mu.Unlock()
	return nil
}

// contextErr 检查 context 是否已取消；nil context 视为配置错误。
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidConfig
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
