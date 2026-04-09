package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Storage 接口
// =============================================================================

// Storage 记忆存储接口
type Storage interface {
	// SaveMessage 保存消息
	SaveMessage(ctx context.Context, msg *ConversationMessage) error
	// GetMessages 获取会话消息
	GetMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error)
	// GetMessageCount 获取会话消息数量
	GetMessageCount(ctx context.Context, userID, sessionID string) (int, error)
	// CleanupMessagesByLimit 清理超限消息（保留最新的 N 条）
	CleanupMessagesByLimit(ctx context.Context, userID, sessionID string, keepCount int) error
	// CleanupMessagesByTime 清理过期消息
	CleanupMessagesByTime(ctx context.Context, olderThan time.Duration) error

	// GetUserMemory 获取用户记忆
	GetUserMemory(ctx context.Context, userID string) (*UserMemory, error)
	// SaveUserMemory 保存用户记忆
	SaveUserMemory(ctx context.Context, memory *UserMemory) error

	// GetSessionSummary 获取会话摘要
	GetSessionSummary(ctx context.Context, userID, sessionID string) (*SessionSummary, error)
	// SaveSessionSummary 保存会话摘要
	SaveSessionSummary(ctx context.Context, summary *SessionSummary) error

	// CleanupOldSessions 清理旧会话
	CleanupOldSessions(ctx context.Context, olderThan time.Duration) error

	// AutoMigrate 自动迁移（用于 SQL 存储）
	AutoMigrate() error
}

// =============================================================================
// MemoryStorage 内存存储实现
// =============================================================================

// MemoryStorage 基于内存的存储实现
// 适用于单实例或测试场景
type MemoryStorage struct {
	mu        sync.RWMutex
	messages  map[string][]*ConversationMessage // key: userID:sessionID
	memories  map[string]*UserMemory            // key: userID
	summaries map[string]*SessionSummary        // key: userID:sessionID

	// 配置
	maxSize    int           // 最大消息数（0 = 无限制）
	windowSize int           // 滑动窗口大小（0 = 不启用）
	ttl        time.Duration // 会话 TTL（0 = 永不过期）
}

// StorageOption 存储配置选项
type StorageOption func(*storageOptions)

type storageOptions struct {
	maxSize    int           // 最大消息数（0 = 无限制）
	ttl        time.Duration // 会话 TTL（0 = 永不过期）
	windowSize int           // 滑动窗口大小（0 = 不启用）
}

// WithMaxSize 设置最大消息数
func WithMaxSize(max int) StorageOption {
	return func(o *storageOptions) {
		o.maxSize = max
	}
}

// WithTTL 设置会话 TTL
func WithTTL(ttl time.Duration) StorageOption {
	return func(o *storageOptions) {
		o.ttl = ttl
	}
}

// WithWindowSize 设置滑动窗口大小
func WithWindowSize(size int) StorageOption {
	return func(o *storageOptions) {
		o.windowSize = size
	}
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(opts ...StorageOption) *MemoryStorage {
	var cfg storageOptions
	for _, opt := range opts {
		opt(&cfg)
	}
	return &MemoryStorage{
		messages:   make(map[string][]*ConversationMessage),
		memories:   make(map[string]*UserMemory),
		summaries:  make(map[string]*SessionSummary),
		maxSize:    cfg.maxSize,
		ttl:        cfg.ttl,
		windowSize: cfg.windowSize,
	}
}

// messageKey 生成消息存储的 key
func messageKey(userID, sessionID string) string {
	return userID + ":" + sessionID
}

// SaveMessage 保存消息
func (s *MemoryStorage) SaveMessage(ctx context.Context, msg *ConversationMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	msg.CreatedAt = time.Now()

	key := messageKey(msg.UserID, msg.SessionID)
	s.messages[key] = append(s.messages[key], msg)

	// 滑动窗口裁剪
	if s.windowSize > 0 && len(s.messages[key]) > s.windowSize {
		s.messages[key] = s.messages[key][len(s.messages[key])-s.windowSize:]
		return nil
	}

	// 最大容量裁剪
	if s.maxSize > 0 && len(s.messages[key]) > s.maxSize {
		s.messages[key] = s.messages[key][len(s.messages[key])-s.maxSize:]
	}

	return nil
}

// GetMessages 获取会话消息
func (s *MemoryStorage) GetMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := messageKey(userID, sessionID)
	msgs := s.messages[key]

	if limit > 0 && len(msgs) > limit {
		return msgs[len(msgs)-limit:], nil
	}
	return msgs, nil
}

// GetMessageCount 获取会话消息数量
func (s *MemoryStorage) GetMessageCount(ctx context.Context, userID, sessionID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := messageKey(userID, sessionID)
	return len(s.messages[key]), nil
}

// CleanupMessagesByLimit 清理超限消息（保留最新的 N 条）
func (s *MemoryStorage) CleanupMessagesByLimit(ctx context.Context, userID, sessionID string, keepCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := messageKey(userID, sessionID)
	if len(s.messages[key]) > keepCount {
		s.messages[key] = s.messages[key][len(s.messages[key])-keepCount:]
	}
	return nil
}

// CleanupMessagesByTime 清理过期消息
func (s *MemoryStorage) CleanupMessagesByTime(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for key, msgs := range s.messages {
		// 检查第一条消息的时间
		if len(msgs) > 0 && msgs[0].CreatedAt.Before(cutoff) {
			delete(s.messages, key)
		}
	}
	return nil
}

// GetUserMemory 获取用户记忆
func (s *MemoryStorage) GetUserMemory(ctx context.Context, userID string) (*UserMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if memory, ok := s.memories[userID]; ok {
		return memory, nil
	}
	return nil, nil
}

// SaveUserMemory 保存用户记忆
func (s *MemoryStorage) SaveUserMemory(ctx context.Context, memory *UserMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	memory.UpdatedAt = now

	s.memories[memory.UserID] = memory
	return nil
}

// GetSessionSummary 获取会话摘要
func (s *MemoryStorage) GetSessionSummary(ctx context.Context, userID, sessionID string) (*SessionSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := messageKey(userID, sessionID)
	if summary, ok := s.summaries[key]; ok {
		return summary, nil
	}
	return nil, nil
}

// SaveSessionSummary 保存会话摘要
func (s *MemoryStorage) SaveSessionSummary(ctx context.Context, summary *SessionSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = now
	}
	summary.UpdatedAt = now

	key := messageKey(summary.UserID, summary.SessionID)
	s.summaries[key] = summary
	return nil
}

// CleanupOldSessions 清理旧会话
func (s *MemoryStorage) CleanupOldSessions(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for key, msgs := range s.messages {
		if len(msgs) > 0 && msgs[0].CreatedAt.Before(cutoff) {
			delete(s.messages, key)
			delete(s.summaries, key)
		}
	}
	return nil
}

// AutoMigrate 自动迁移（内存存储无需操作）
func (s *MemoryStorage) AutoMigrate() error {
	return nil
}

// =============================================================================
// 工具函数
// =============================================================================

// MessagesToText 将消息列表转换为文本
func MessagesToText(msgs []*ConversationMessage) string {
	if len(msgs) == 0 {
		return ""
	}

	var text string
	for _, msg := range msgs {
		role := msg.Role
		if role == "" {
			role = "user"
		}
		text += role + ": " + msg.Content + "\n\n"
	}
	return text
}
