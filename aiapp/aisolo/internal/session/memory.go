package session

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"zero-service/aiapp/aisolo/aisolo"
)

// MemoryStore 进程内 Session 存储, 仅用于开发/测试。生产请改用持久化后端。
type MemoryStore struct {
	mu         sync.RWMutex
	sessions   map[string]*Session         // key = userID + ":" + sessionID
	interrupts map[string]*InterruptRecord // key = interruptID
}

// NewMemoryStore 构造 MemoryStore。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions:   make(map[string]*Session),
		interrupts: make(map[string]*InterruptRecord),
	}
}

// NewStore 目前只支持 memory; 保留工厂以便后续扩展。
func NewStore(cfg Config) (Store, error) {
	switch cfg.Type {
	case "", "memory":
		return NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("session: unsupported store type %q", cfg.Type)
	}
}

func sessionKey(userID, sessionID string) string { return userID + ":" + sessionID }

func (s *MemoryStore) CreateSession(_ context.Context, sess *Session) error {
	if sess == nil || sess.ID == "" || sess.UserID == "" {
		return fmt.Errorf("session: empty id/user")
	}
	k := sessionKey(sess.UserID, sess.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[k]; ok {
		return fmt.Errorf("session: %q already exists", sess.ID)
	}
	cp := *sess
	s.sessions[k] = &cp
	return nil
}

func (s *MemoryStore) GetSession(_ context.Context, userID, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionKey(userID, sessionID)]
	if !ok {
		return nil, fmt.Errorf("session: not found")
	}
	cp := *sess
	return &cp, nil
}

func (s *MemoryStore) UpdateSession(_ context.Context, sess *Session) error {
	if sess == nil || sess.ID == "" {
		return fmt.Errorf("session: empty")
	}
	k := sessionKey(sess.UserID, sess.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[k]; !ok {
		return fmt.Errorf("session: not found")
	}
	cp := *sess
	s.sessions[k] = &cp
	return nil
}

func (s *MemoryStore) ListSessions(_ context.Context, userID string, page, pageSize int) ([]*Session, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*Session
	for _, sess := range s.sessions {
		if sess.UserID != userID {
			continue
		}
		cp := *sess
		list = append(list, &cp)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
	total := int64(len(list))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(list) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(list) {
		end = len(list)
	}
	return list[start:end], total, nil
}

func (s *MemoryStore) DeleteSession(_ context.Context, userID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := sessionKey(userID, sessionID)
	if _, ok := s.sessions[k]; !ok {
		return fmt.Errorf("session: not found")
	}
	delete(s.sessions, k)
	return nil
}

func (s *MemoryStore) SaveInterrupt(_ context.Context, r *InterruptRecord) error {
	if r == nil || r.InterruptID == "" {
		return fmt.Errorf("session: empty interrupt")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *r
	s.interrupts[r.InterruptID] = &cp
	return nil
}

func (s *MemoryStore) GetInterrupt(_ context.Context, id string) (*InterruptRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.interrupts[id]
	if !ok {
		return nil, fmt.Errorf("session: interrupt %q not found", id)
	}
	cp := *r
	return &cp, nil
}

func (s *MemoryStore) ResetRunningToIdle(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, sess := range s.sessions {
		if sess.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
			sess.Status = aisolo.SessionStatus_SESSION_STATUS_IDLE
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) Close() error { return nil }
