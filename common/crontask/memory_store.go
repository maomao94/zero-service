package crontask

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

var _ TaskStore = (*MemoryStore)(nil)

// MemoryStore TaskStore 的内存实现，适用于单实例或开发测试场景。
// LockAndFetch 参照 trigger 扫表逻辑：同优先级随机选一条，立即时间扩展防并发。
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]*TaskConfig
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]*TaskConfig),
	}
}

// LockAndFetch 从内存中扫描并锁定一个到期任务。
// 按 priority DESC 排序，同优先级随机选一条。
// 选中后立即 nextRun = now+实际锁超时（时间扩展）。
func (m *MemoryStore) LockAndFetch(ctx context.Context, now time.Time, defaultLockTimeout time.Duration) (*TaskClaim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var candidates []*TaskConfig
	for _, t := range m.tasks {
		if t.Status == StatusEnabled && !t.NextRun.IsZero() && !t.NextRun.After(now) {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return nil, ErrNotFound
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})

	top := candidates[0]
	last := 0
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Priority == top.Priority {
			last = i
		} else {
			break
		}
	}
	task := candidates[rand.Intn(last+1)]

	originalNextRun := task.NextRun
	lockTimeout := ResolveLockTimeout(task.LockTimeout, defaultLockTimeout)
	task.NextRun = now.Add(lockTimeout).Truncate(time.Second)

	claimedTask := *task
	claimedTask.NextRun = originalNextRun
	return &TaskClaim{Task: &claimedTask, LockedUntil: task.NextRun}, nil
}

// Complete 使用 LockedUntil token 完成一次周期执行。
func (m *MemoryStore) Complete(ctx context.Context, id string, expectedLockedUntil, nextRun, lastRun time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok || !task.NextRun.Equal(expectedLockedUntil) {
		return ErrNotFound
	}
	task.NextRun = nextRun
	if !lastRun.IsZero() {
		task.LastRun = lastRun
	}
	return nil
}

// UpdateLastRun 只记录手动执行成功时间。
func (m *MemoryStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	task.LastRun = lastRun
	return nil
}

// GetByCode 按全局唯一的 task_code 查询任务。
func (m *MemoryStore) GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tasks {
		if t.TaskCode == taskCode {
			task := *t
			return &task, nil
		}
	}
	return nil, ErrNotFound
}

// Insert 新增任务，自动分配 ID，task_code 冲突返回 ErrDuplicate。
func (m *MemoryStore) Insert(ctx context.Context, cfg *TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ValidateRRule(cfg.RRuleStr); err != nil {
		return err
	}

	if cfg.ID == "" {
		for {
			cfg.ID = fmt.Sprintf("auto-%d", time.Now().UnixNano())
			if _, exists := m.tasks[cfg.ID]; !exists {
				break
			}
		}
	}

	for _, t := range m.tasks {
		if t.TaskCode == cfg.TaskCode {
			return ErrDuplicate
		}
	}

	task := *cfg
	m.tasks[cfg.ID] = &task
	return nil
}

// Update 按 id 全量更新任务，task_code 与其他记录冲突时返回 ErrDuplicate。
func (m *MemoryStore) Update(ctx context.Context, cfg *TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ValidateRRule(cfg.RRuleStr); err != nil {
		return err
	}

	for _, t := range m.tasks {
		if t.ID != cfg.ID && t.TaskCode == cfg.TaskCode {
			return ErrDuplicate
		}
	}

	if _, ok := m.tasks[cfg.ID]; !ok {
		return ErrNotFound
	}

	lastRun := m.tasks[cfg.ID].LastRun
	updated := *cfg
	updated.LastRun = lastRun
	m.tasks[cfg.ID] = &updated
	return nil
}

// Enable 启用任务，并从当前时间重新计算未来 NextRun。
func (m *MemoryStore) Enable(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	if task.Status == StatusEnabled {
		return nil
	}
	nextRun, err := NextAfter(task.RRuleStr, time.Now())
	if err != nil {
		return err
	}
	task.Status = StatusEnabled
	task.NextRun = nextRun
	return nil
}

// Disable 禁用任务，不撤销已经 claim 的在途执行。
func (m *MemoryStore) Disable(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return ErrUpdate
	}
	task.Status = StatusDisabled
	return nil
}

// Delete 删除任务。
func (m *MemoryStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

// List 按条件获取任务；零值条件返回全部任务。
func (m *MemoryStore) List(ctx context.Context, condition ListCondition) ([]*TaskConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*TaskConfig
	for _, t := range m.tasks {
		if len(condition.Statuses) > 0 && !containsStatus(condition.Statuses, t.Status) {
			continue
		}
		task := *t
		result = append(result, &task)
	}
	return result, nil
}

func containsStatus(statuses []TaskStatus, target TaskStatus) bool {
	for _, status := range statuses {
		if status == target {
			return true
		}
	}
	return false
}
