package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JSONLStorage 基于本地 JSONL 文件的存储实现。
//
// 文件布局：BaseDir/<userID>/<sessionID>.jsonl，每行一条 ConversationMessage。
// 适合单机部署、简单持久化的场景（借鉴 eino-examples/quickstart/chatwitheino/mem）。
type JSONLStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewJSONLStorage 创建 JSONL 存储。
func NewJSONLStorage(baseDir string) (*JSONLStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("memory.jsonl: mkdir %s: %w", baseDir, err)
	}
	return &JSONLStorage{baseDir: baseDir}, nil
}

func (s *JSONLStorage) sessionFile(userID, sessionID string) string {
	return filepath.Join(s.baseDir, sanitizeFilename(userID), sanitizeFilename(sessionID)+".jsonl")
}

// SaveMessage 追加一条消息到 .jsonl 文件。
func (s *JSONLStorage) SaveMessage(_ context.Context, msg *ConversationMessage) error {
	if msg == nil {
		return nil
	}
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fp := s.sessionFile(msg.UserID, msg.SessionID)
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Errorf("memory.jsonl: mkdir %s: %w", filepath.Dir(fp), err)
	}
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("memory.jsonl: open %s: %w", fp, err)
	}
	defer f.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("memory.jsonl: marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("memory.jsonl: write: %w", err)
	}
	return nil
}

// GetMessages 读取会话文件并按 CreatedAt 升序返回。
func (s *JSONLStorage) GetMessages(_ context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fp := s.sessionFile(userID, sessionID)
	f, err := os.Open(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("memory.jsonl: open %s: %w", fp, err)
	}
	defer f.Close()

	var msgs []*ConversationMessage
	scanner := bufio.NewScanner(f)
	// 大消息可能超过默认 64KB
	scanner.Buffer(make([]byte, 1<<20), 16<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg ConversationMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			return nil, fmt.Errorf("memory.jsonl: unmarshal: %w", err)
		}
		msgs = append(msgs, &msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("memory.jsonl: scan: %w", err)
	}

	sortMessagesAsc(msgs)
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

// DeleteSession 删除会话对应的 jsonl 文件。
func (s *JSONLStorage) DeleteSession(_ context.Context, userID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fp := s.sessionFile(userID, sessionID)
	if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("memory.jsonl: remove %s: %w", fp, err)
	}
	return nil
}

// Close 无资源需要释放。
func (s *JSONLStorage) Close() error {
	return nil
}
