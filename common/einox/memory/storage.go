package memory

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// Storage 接口定义
// =============================================================================

// Storage 记忆存储接口
type Storage interface {
	// Save 保存消息
	Save(ctx context.Context, sessionID string, msg *schema.Message) error
	// Get 获取会话消息（按时间倒序）
	Get(ctx context.Context, sessionID string, limit int) ([]*schema.Message, error)
	// Clear 清除会话
	Clear(ctx context.Context, sessionID string) error
	// Count 会话消息数量
	Count(ctx context.Context, sessionID string) (int, error)
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

// =============================================================================
// MemoryStorage 内存存储实现
// =============================================================================

// MemoryStorage 基于内存的存储实现
type MemoryStorage struct {
	mu       sync.RWMutex
	sessions map[string][]*schema.Message
	opts     storageOptions
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(opts ...StorageOption) *MemoryStorage {
	var cfg storageOptions
	for _, opt := range opts {
		opt(&cfg)
	}
	return &MemoryStorage{
		sessions: make(map[string][]*schema.Message),
		opts:     cfg,
	}
}

func (s *MemoryStorage) Save(ctx context.Context, sessionID string, msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = append(s.sessions[sessionID], msg)

	// 滑动窗口裁剪
	if s.opts.windowSize > 0 && len(s.sessions[sessionID]) > s.opts.windowSize {
		s.sessions[sessionID] = s.sessions[sessionID][len(s.sessions[sessionID])-s.opts.windowSize:]
		return nil
	}

	// 最大容量裁剪
	if s.opts.maxSize > 0 && len(s.sessions[sessionID]) > s.opts.maxSize {
		s.sessions[sessionID] = s.sessions[sessionID][len(s.sessions[sessionID])-s.opts.maxSize:]
	}

	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, sessionID string, limit int) ([]*schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.sessions[sessionID]
	if limit > 0 && len(msgs) > limit {
		return msgs[len(msgs)-limit:], nil
	}
	return msgs, nil
}

func (s *MemoryStorage) Clear(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *MemoryStorage) Count(ctx context.Context, sessionID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions[sessionID]), nil
}
