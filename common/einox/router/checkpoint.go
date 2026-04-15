package router

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// =============================================================================
// CheckPointStore 接口
// =============================================================================

// InterruptType 中断类型
type InterruptType string

const (
	InterruptTypeUserCancel    InterruptType = "user_cancel"     // 用户主动取消
	InterruptTypeNeedUserInput InterruptType = "need_user_input" // 需要用户输入/选择
	InterruptTypeError         InterruptType = "error"           // 执行错误中断
	InterruptTypeTimeout       InterruptType = "timeout"         // 超时中断
	InterruptTypeRateLimit     InterruptType = "rate_limit"      // 限流中断
)

// CheckPoint 执行检查点
type CheckPoint struct {
	ID            string                 `json:"id"`             // 检查点ID
	SessionID     string                 `json:"session_id"`     // 会话ID
	UserID        string                 `json:"user_id"`        // 用户ID
	InterruptType InterruptType          `json:"interrupt_type"` // 中断类型
	Status        string                 `json:"status"`         // 状态: running/paused/finished/failed
	RetryCount    int                    `json:"retry_count"`    // 重试次数
	MaxRetry      int                    `json:"max_retry"`      // 最大重试次数
	ExpireAt      time.Time              `json:"expire_at"`      // 过期时间
	State         map[string]interface{} `json:"state"`          // 执行状态数据
	Error         string                 `json:"error"`          // 错误信息（如果是错误中断）
	CreatedAt     time.Time              `json:"created_at"`     // 创建时间
	UpdatedAt     time.Time              `json:"updated_at"`     // 更新时间
}

// CheckPointStore 检查点存储接口
// 用于中断恢复时保存和恢复 Agent 执行状态
type CheckPointStore interface {
	// Set 保存检查点
	Set(ctx context.Context, checkPointID string, checkPoint []byte) error

	// Get 获取检查点
	// 返回值: (检查点数据, 是否存在, 错误)
	Get(ctx context.Context, checkPointID string) ([]byte, bool, error)

	// Delete 删除检查点
	Delete(ctx context.Context, checkPointID string) error

	// Exists 检查点是否存在
	Exists(ctx context.Context, checkPointID string) (bool, error)

	// ListBySession 获取会话下的所有检查点
	ListBySession(ctx context.Context, sessionID string) ([]*CheckPoint, error)

	// ListByUser 获取用户下的所有检查点
	ListByUser(ctx context.Context, userID string) ([]*CheckPoint, error)

	// CleanupExpired 清理过期的检查点
	CleanupExpired(ctx context.Context) (int, error)

	// UpdateStatus 更新检查点状态
	UpdateStatus(ctx context.Context, checkPointID string, status string, errMsg string) error

	// IncrementRetry 增加重试次数
	IncrementRetry(ctx context.Context, checkPointID string) (int, error)
}

// =============================================================================
// MemoryCheckPointStore 内存存储（开发用）
// =============================================================================

// MemoryCheckPointStore 内存实现的检查点存储
// 适用于单实例部署，生产环境建议使用 Redis
type MemoryCheckPointStore struct {
	mu   sync.RWMutex
	data map[string]*memoryCheckpoint
	ttl  time.Duration // 生存时间，0 表示永不过期
}

// memoryCheckpoint 内存检查点
type memoryCheckpoint struct {
	data      []byte
	createdAt time.Time
	expiresAt time.Time
}

// NewMemoryCheckPointStore 创建内存检查点存储
func NewMemoryCheckPointStore(ttl time.Duration) *MemoryCheckPointStore {
	return &MemoryCheckPointStore{
		data: make(map[string]*memoryCheckpoint),
		ttl:  ttl,
	}
}

// Set 保存检查点
func (s *MemoryCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiresAt := time.Time{}
	if s.ttl > 0 {
		expiresAt = now.Add(s.ttl)
	}

	s.data[checkPointID] = &memoryCheckpoint{
		data:      checkPoint,
		createdAt: now,
		expiresAt: expiresAt,
	}

	return nil
}

// Get 获取检查点
func (s *MemoryCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.data[checkPointID]
	if !ok {
		return nil, false, nil
	}

	// 检查是否过期
	if !cp.expiresAt.IsZero() && time.Now().After(cp.expiresAt) {
		return nil, false, nil
	}

	// 返回数据副本
	data := make([]byte, len(cp.data))
	copy(data, cp.data)
	return data, true, nil
}

// Delete 删除检查点
func (s *MemoryCheckPointStore) Delete(ctx context.Context, checkPointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, checkPointID)
	return nil
}

// Exists 检查点是否存在
func (s *MemoryCheckPointStore) Exists(ctx context.Context, checkPointID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.data[checkPointID]
	if !ok {
		return false, nil
	}

	// 检查是否过期
	if !cp.expiresAt.IsZero() && time.Now().After(cp.expiresAt) {
		return false, nil
	}

	return true, nil
}

// Size 返回存储的检查点数量
func (s *MemoryCheckPointStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Clear 清除所有检查点
func (s *MemoryCheckPointStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*memoryCheckpoint)
}

// Cleanup 清理过期检查点
func (s *MemoryCheckPointStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	for id, cp := range s.data {
		if !cp.expiresAt.IsZero() && now.After(cp.expiresAt) {
			delete(s.data, id)
			count++
		}
	}
	return count
}

// StartCleanupDaemon 启动清理守护进程
// 每隔 interval 清理一次过期检查点
func (s *MemoryCheckPointStore) StartCleanupDaemon(interval time.Duration) func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-stop:
				return
			}
		}
	}()

	return func() { close(stop) }
}

// ListBySession 获取会话下的所有检查点
func (s *MemoryCheckPointStore) ListBySession(ctx context.Context, sessionID string) ([]*CheckPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*CheckPoint
	now := time.Now()
	for _, cp := range s.data {
		var checkPoint CheckPoint
		if err := json.Unmarshal(cp.data, &checkPoint); err != nil {
			continue
		}
		// 过滤未过期且会话匹配的
		if checkPoint.SessionID == sessionID && (cp.expiresAt.IsZero() || now.Before(cp.expiresAt)) {
			results = append(results, &checkPoint)
		}
	}
	return results, nil
}

// ListByUser 获取用户下的所有检查点
func (s *MemoryCheckPointStore) ListByUser(ctx context.Context, userID string) ([]*CheckPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*CheckPoint
	now := time.Now()
	for _, cp := range s.data {
		var checkPoint CheckPoint
		if err := json.Unmarshal(cp.data, &checkPoint); err != nil {
			continue
		}
		// 过滤未过期且用户匹配的
		if checkPoint.UserID == userID && (cp.expiresAt.IsZero() || now.Before(cp.expiresAt)) {
			results = append(results, &checkPoint)
		}
	}
	return results, nil
}

// CleanupExpired 清理过期的检查点
func (s *MemoryCheckPointStore) CleanupExpired(ctx context.Context) (int, error) {
	return s.Cleanup(), nil
}

// UpdateStatus 更新检查点状态
func (s *MemoryCheckPointStore) UpdateStatus(ctx context.Context, checkPointID string, status string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.data[checkPointID]
	if !ok {
		return nil
	}

	var checkPoint CheckPoint
	if err := json.Unmarshal(cp.data, &checkPoint); err != nil {
		return err
	}

	checkPoint.Status = status
	checkPoint.Error = errMsg
	checkPoint.UpdatedAt = time.Now()

	newData, err := json.Marshal(checkPoint)
	if err != nil {
		return err
	}

	cp.data = newData
	return nil
}

// IncrementRetry 增加重试次数
func (s *MemoryCheckPointStore) IncrementRetry(ctx context.Context, checkPointID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.data[checkPointID]
	if !ok {
		return 0, nil
	}

	var checkPoint CheckPoint
	if err := json.Unmarshal(cp.data, &checkPoint); err != nil {
		return 0, err
	}

	checkPoint.RetryCount++
	checkPoint.UpdatedAt = time.Now()

	newData, err := json.Marshal(checkPoint)
	if err != nil {
		return 0, err
	}

	cp.data = newData
	return checkPoint.RetryCount, nil
}

// =============================================================================
// NoOpCheckPointStore 无操作存储
// =============================================================================

// NoOpCheckPointStore 不存储任何数据的检查点存储
// 适用于不需要中断恢复的场景
type NoOpCheckPointStore struct{}

// NewNoOpCheckPointStore 创建无操作存储
func NewNoOpCheckPointStore() *NoOpCheckPointStore {
	return &NoOpCheckPointStore{}
}

func (s *NoOpCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return nil
}

func (s *NoOpCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	return nil, false, nil
}

func (s *NoOpCheckPointStore) Delete(ctx context.Context, checkPointID string) error {
	return nil
}

func (s *NoOpCheckPointStore) Exists(ctx context.Context, checkPointID string) (bool, error) {
	return false, nil
}

// ListBySession 获取会话下的所有检查点
func (s *NoOpCheckPointStore) ListBySession(ctx context.Context, sessionID string) ([]*CheckPoint, error) {
	return nil, nil
}

// ListByUser 获取用户下的所有检查点
func (s *NoOpCheckPointStore) ListByUser(ctx context.Context, userID string) ([]*CheckPoint, error) {
	return nil, nil
}

// CleanupExpired 清理过期的检查点
func (s *NoOpCheckPointStore) CleanupExpired(ctx context.Context) (int, error) {
	return 0, nil
}

// UpdateStatus 更新检查点状态
func (s *NoOpCheckPointStore) UpdateStatus(ctx context.Context, checkPointID string, status string, errMsg string) error {
	return nil
}

// IncrementRetry 增加重试次数
func (s *NoOpCheckPointStore) IncrementRetry(ctx context.Context, checkPointID string) (int, error) {
	return 0, nil
}

// =============================================================================
// 全局检查点存储管理器
// =============================================================================

var (
	globalStore   CheckPointStore
	globalStoreMu sync.RWMutex
)

// SetGlobalCheckPointStore 设置全局检查点存储
func SetGlobalCheckPointStore(store CheckPointStore) {
	globalStoreMu.Lock()
	defer globalStoreMu.Unlock()
	globalStore = store
}

// GetGlobalCheckPointStore 获取全局检查点存储
func GetGlobalCheckPointStore() CheckPointStore {
	globalStoreMu.RLock()
	defer globalStoreMu.RUnlock()
	return globalStore
}

// InitDefaultCheckPointStore 初始化默认检查点存储
func InitDefaultCheckPointStore(ttl time.Duration) {
	SetGlobalCheckPointStore(NewMemoryCheckPointStore(ttl))
}
