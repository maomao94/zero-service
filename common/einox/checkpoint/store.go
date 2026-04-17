// Package checkpoint 为 eino adk Runner 提供可切换后端的 CheckPointStore 实现。
//
// 后端类型：
//   - memory：进程内存，适合单实例 / 测试
//   - jsonl：每 key 一个 .jsonl 文件，适合单机持久化
//   - gormx：关系型数据库，适合分布式部署
//
// 所有实现均满足 adk.CheckPointStore 接口（Set/Get），使三种后端可以无缝互换，
// 和 common/einox/memory、aisolo/internal/store 的多后端设计保持一致。
package checkpoint

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/adk"

	"zero-service/common/gormx"
)

// Store 在 adk.CheckPointStore 之上追加 Delete 与 Close，方便业务侧清理与释放资源。
type Store interface {
	adk.CheckPointStore
	Delete(ctx context.Context, key string) error
	Close() error
}

// Type 存储类型枚举。
type Type string

const (
	TypeMemory Type = "memory"
	TypeJSONL  Type = "jsonl"
	TypeGORMX  Type = "gormx"
)

// Config checkpoint 存储配置。gormx 由调用方传入 *gormx.DB 而非 DSN，避免重复建连。
type Config struct {
	Type    Type   `json:",default=memory,options=memory|jsonl|gormx"`
	BaseDir string `json:",optional"` // JSONL 根目录
}

// NewStore 按配置构造 Store。gormx 类型需同时提供 *gormx.DB（为空时降级到 memory）。
func NewStore(cfg Config, db *gormx.DB) (Store, error) {
	switch cfg.Type {
	case "", TypeMemory:
		return NewMemoryStore(), nil
	case TypeJSONL:
		if cfg.BaseDir == "" {
			return nil, fmt.Errorf("checkpoint.NewStore: jsonl.BaseDir is required")
		}
		return NewJSONLStore(cfg.BaseDir)
	case TypeGORMX:
		if db == nil {
			return nil, fmt.Errorf("checkpoint.NewStore: gormx requires *gormx.DB")
		}
		return NewGormxStore(db)
	default:
		return nil, fmt.Errorf("checkpoint.NewStore: unknown type %q", cfg.Type)
	}
}

// =============================================================================
// MemoryStore 内存实现
// =============================================================================

// MemoryStore 线程安全的内存 checkpoint 存储。
type MemoryStore struct {
	mu sync.RWMutex
	m  map[string][]byte
}

// NewMemoryStore 创建内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{m: make(map[string][]byte)}
}

// Set 保存快照。内部复制一份，避免调用方后续修改原切片。
func (s *MemoryStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = append([]byte(nil), value...)
	return nil
}

// Get 读取快照。
func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), v...), true, nil
}

// Delete 删除快照。
func (s *MemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
	return nil
}

// Close 无资源需要释放。
func (s *MemoryStore) Close() error { return nil }

var _ Store = (*MemoryStore)(nil)
