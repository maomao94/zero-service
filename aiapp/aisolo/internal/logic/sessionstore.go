package logic

import (
	"context"
	"sync"
	"time"
	"zero-service/aiapp/aisolo/aisolo"

	"github.com/google/uuid"
)

// SessionStore 会话存储接口
type SessionStore interface {
	Create(ctx context.Context, userID string) (*aisolo.Session, error)
	Get(ctx context.Context, sessionID string) (*aisolo.Session, error)
	List(ctx context.Context, userID string, page, size int) ([]*aisolo.Session, int64, error)
	Delete(ctx context.Context, sessionID string) error
}

// InMemorySessionStore 内存会话存储
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*aisolo.Session
}

// NewInMemorySessionStore 创建内存会话存储
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*aisolo.Session),
	}
}

// Create 创建会话
func (s *InMemorySessionStore) Create(ctx context.Context, userID string) (*aisolo.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	sessionID := uuid.NewString()

	session := &aisolo.Session{
		SessionId:    sessionID,
		UserId:       userID,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
	}

	s.sessions[sessionID] = session
	return session, nil
}

// Get 获取会话
func (s *InMemorySessionStore) Get(ctx context.Context, sessionID string) (*aisolo.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if session, ok := s.sessions[sessionID]; ok {
		return session, nil
	}
	return nil, nil
}

// List 列出会话
func (s *InMemorySessionStore) List(ctx context.Context, userID string, page, size int) ([]*aisolo.Session, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*aisolo.Session
	for _, session := range s.sessions {
		if session.UserId == userID {
			sessions = append(sessions, session)
		}
	}

	// 分页
	total := int64(len(sessions))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	start := (page - 1) * size
	end := start + size

	if start >= len(sessions) {
		return []*aisolo.Session{}, total, nil
	}
	if end > len(sessions) {
		end = len(sessions)
	}

	return sessions[start:end], total, nil
}

// Delete 删除会话
func (s *InMemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// GlobalSessionStore 全局会话存储实例
var GlobalSessionStore = NewInMemorySessionStore()
