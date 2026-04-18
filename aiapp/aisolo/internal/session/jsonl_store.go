package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
)

// JSONLStore 单机目录持久化：每会话一个 JSON 文件 + 每条中断一个 JSON。
// 适合开发或小流量；多实例请勿使用 jsonl 会话存储（无跨节点协调）。
type JSONLStore struct {
	base                  string
	mu                    sync.Mutex
	nullLeaseRecoverGrace time.Duration
}

// NewJSONLStore 创建 jsonl 目录存储；baseDir 不可为空。
func NewJSONLStore(baseDir string, nullLeaseRecoverGrace time.Duration) (*JSONLStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("session.jsonl: empty baseDir")
	}
	if nullLeaseRecoverGrace <= 0 {
		nullLeaseRecoverGrace = 2 * time.Minute
	}
	for _, sub := range []string{"sessions", "interrupts"} {
		if err := os.MkdirAll(filepath.Join(baseDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("session.jsonl: mkdir: %w", err)
		}
	}
	return &JSONLStore{base: baseDir, nullLeaseRecoverGrace: nullLeaseRecoverGrace}, nil
}

func (s *JSONLStore) sessPath(userID, sessionID string) string {
	return filepath.Join(s.base, "sessions", sanitizePath(userID)+"_"+sanitizePath(sessionID)+".json")
}

func (s *JSONLStore) intPath(interruptID string) string {
	return filepath.Join(s.base, "interrupts", sanitizePath(interruptID)+".json")
}

func sanitizePath(p string) string {
	return strings.ReplaceAll(strings.ReplaceAll(p, "/", "_"), "\\", "_")
}

func (s *JSONLStore) CreateSession(ctx context.Context, sess *Session) error {
	_ = ctx
	if sess == nil || sess.ID == "" || sess.UserID == "" {
		return fmt.Errorf("session.jsonl: empty id/user")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fp := s.sessPath(sess.UserID, sess.ID)
	if _, err := os.Stat(fp); err == nil {
		return fmt.Errorf("session: %q already exists", sess.ID)
	}
	return writeJSON(fp, sess)
}

func (s *JSONLStore) GetSession(ctx context.Context, userID, sessionID string) (*Session, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	var sess Session
	if err := readJSON(s.sessPath(userID, sessionID), &sess); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session: not found")
		}
		return nil, err
	}
	return &sess, nil
}

func (s *JSONLStore) UpdateSession(ctx context.Context, sess *Session) error {
	_ = ctx
	if sess == nil || sess.ID == "" {
		return fmt.Errorf("session.jsonl: empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fp := s.sessPath(sess.UserID, sess.ID)
	if _, err := os.Stat(fp); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session: not found")
		}
		return err
	}
	return writeJSON(fp, sess)
}

func (s *JSONLStore) ListSessions(ctx context.Context, userID string, page, pageSize int) ([]*Session, int64, error) {
	_ = ctx
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.base, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	prefix := sanitizePath(userID) + "_"
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	var list []*Session
	for _, p := range paths {
		var sess Session
		if err := readJSON(p, &sess); err != nil {
			continue
		}
		if sess.UserID == userID {
			cp := sess
			list = append(list, &cp)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
	total := int64(len(list))
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

func (s *JSONLStore) DeleteSession(ctx context.Context, userID, sessionID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	fp := s.sessPath(userID, sessionID)
	if _, err := os.Stat(fp); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session: not found")
		}
		return err
	}
	_ = s.deleteInterruptsForSessionLocked(userID, sessionID)
	return os.Remove(fp)
}

func (s *JSONLStore) deleteInterruptsForSessionLocked(userID, sessionID string) error {
	dir := filepath.Join(s.base, "interrupts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		var rec InterruptRecord
		if err := readJSON(p, &rec); err != nil {
			continue
		}
		if rec.SessionID == sessionID && rec.UserID == userID {
			_ = os.Remove(p)
		}
	}
	return nil
}

func (s *JSONLStore) SaveInterrupt(ctx context.Context, r *InterruptRecord) error {
	_ = ctx
	if r == nil || r.InterruptID == "" {
		return fmt.Errorf("session.jsonl: empty interrupt")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSON(s.intPath(r.InterruptID), r)
}

func (s *JSONLStore) GetInterrupt(ctx context.Context, id string) (*InterruptRecord, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	var rec InterruptRecord
	if err := readJSON(s.intPath(id), &rec); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session: interrupt %q not found", id)
		}
		return nil, err
	}
	return &rec, nil
}

func (s *JSONLStore) RecoverRunningSessions(ctx context.Context) (int, error) {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.base, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		var sess Session
		if err := readJSON(p, &sess); err != nil {
			continue
		}
		if sess.Status != aisolo.SessionStatus_SESSION_STATUS_RUNNING {
			continue
		}
		if !LeaseStaleForRecover(&sess, now, s.nullLeaseRecoverGrace) {
			continue
		}
		sess.Status = aisolo.SessionStatus_SESSION_STATUS_IDLE
		sess.RunOwner = ""
		sess.RunLeaseUntil = time.Time{}
		sess.UpdatedAt = now
		if err := writeJSON(p, &sess); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *JSONLStore) Close() error { return nil }

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

var _ Store = (*JSONLStore)(nil)
