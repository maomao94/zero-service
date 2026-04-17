// Package memory 提供对话消息的持久化存储。
//
// 定义了最小的 Storage 接口（SaveMessage/GetMessages/DeleteSession），
// 以及三种可切换的实现：memory / jsonl / gormx。
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Storage 接口
// =============================================================================

// Storage 对话消息存储接口。
//
// 设计原则：
//   - 仅包含最小必要的消息持久化能力
//   - 其他增强能力（用户记忆、会话摘要等）不属于核心存储职责
type Storage interface {
	// SaveMessage 保存一条对话消息。
	SaveMessage(ctx context.Context, msg *ConversationMessage) error

	// GetMessages 读取指定会话的消息（按时间升序）。
	//   limit <= 0 表示返回全部，>0 时返回最新的 limit 条。
	GetMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error)

	// DeleteSession 删除指定会话的全部消息。
	DeleteSession(ctx context.Context, userID, sessionID string) error

	// Close 关闭存储（释放文件/数据库等资源）。
	Close() error
}

// =============================================================================
// 工厂
// =============================================================================

// Type 存储类型。
type Type string

const (
	TypeMemory Type = "memory" // 内存存储（默认）
	TypeJSONL  Type = "jsonl"  // JSONL 文件存储
	TypeGORMX  Type = "gormx"  // 基于 common/gormx 的关系型数据库存储
)

// Config 存储配置。
//
// 对于 gormx 类型，调用方需通过 NewStorageWithGormxDB 直接注入 *gormx.DB，
// 这里的 Config 不持有数据库连接，避免把基础设施细节耦合到公共库。
type Config struct {
	Type Type `json:",default=memory,options=memory|jsonl|gormx"`

	// JSONL 配置
	BaseDir string `json:",optional"` // JSONL 根目录，每个会话一个 .jsonl 文件
}

// NewStorage 根据配置构造非 gormx 类型的 Storage 实例。
//
// gormx 类型请使用 NewGormxStorage 直接传入 *gormx.DB。
func NewStorage(cfg Config) (Storage, error) {
	switch cfg.Type {
	case "", TypeMemory:
		return NewMemoryStorage(), nil
	case TypeJSONL:
		if cfg.BaseDir == "" {
			return nil, fmt.Errorf("memory.NewStorage: jsonl.BaseDir is required")
		}
		return NewJSONLStorage(cfg.BaseDir)
	case TypeGORMX:
		return nil, fmt.Errorf("memory.NewStorage: gormx type requires NewGormxStorage(db); pass *gormx.DB directly")
	default:
		return nil, fmt.Errorf("memory.NewStorage: unknown type %q", cfg.Type)
	}
}

// =============================================================================
// MemoryStorage 内存实现
// =============================================================================

// MemoryStorage 基于内存的存储实现，适用于单实例或测试场景。
type MemoryStorage struct {
	mu       sync.RWMutex
	messages map[string][]*ConversationMessage // key: userID:sessionID
}

// NewMemoryStorage 创建内存存储。
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		messages: make(map[string][]*ConversationMessage),
	}
}

func messageKey(userID, sessionID string) string {
	return userID + ":" + sessionID
}

// SaveMessage 保存消息
func (s *MemoryStorage) SaveMessage(_ context.Context, msg *ConversationMessage) error {
	if msg == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	key := messageKey(msg.UserID, msg.SessionID)
	s.messages[key] = append(s.messages[key], msg)
	return nil
}

// GetMessages 获取会话消息
func (s *MemoryStorage) GetMessages(_ context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.messages[messageKey(userID, sessionID)]
	if limit > 0 && len(msgs) > limit {
		return append([]*ConversationMessage(nil), msgs[len(msgs)-limit:]...), nil
	}
	return append([]*ConversationMessage(nil), msgs...), nil
}

// DeleteSession 删除会话的全部消息
func (s *MemoryStorage) DeleteSession(_ context.Context, userID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messages, messageKey(userID, sessionID))
	return nil
}

// Close 无资源需要释放。
func (s *MemoryStorage) Close() error {
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// sortMessagesAsc 按 CreatedAt 升序排序。
func sortMessagesAsc(msgs []*ConversationMessage) {
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})
}

// sanitizeFilename 清理不合法的文件名字符。
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/', r == '\\', r == ':', r == '*', r == '?', r == '"', r == '<', r == '>', r == '|':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
