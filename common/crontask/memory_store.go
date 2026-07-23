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
// 选中后立即 nextRun = now+lockDur（时间扩展），version++。
func (m *MemoryStore) LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*TaskConfig, error) {
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
	task.NextRun = now.Add(lockDur)
	task.Version++

	cloned := cloneTaskConfig(task)
	cloned.NextRun = originalNextRun
	return cloned, nil
}

// UpdateNextRun 更新下次调度时间和上次执行时间。
func (m *MemoryStore) UpdateNextRun(ctx context.Context, id string, nextRun, lastRun time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	task.NextRun = nextRun
	task.LastRun = lastRun
	return nil
}

// GetByCode 按全局唯一的 task_code 查询任务。
func (m *MemoryStore) GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tasks {
		if t.TaskCode == taskCode {
			return cloneTaskConfig(t), nil
		}
	}
	return nil, ErrNotFound
}

// Insert 新增任务，自动分配 ID，task_code 冲突返回 ErrDuplicate。
func (m *MemoryStore) Insert(ctx context.Context, cfg *TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.ID == "" {
		cfg.ID = fmt.Sprintf("auto-%d", time.Now().UnixNano())
	}

	for _, t := range m.tasks {
		if t.TaskCode == cfg.TaskCode {
			return ErrDuplicate
		}
	}

	m.tasks[cfg.ID] = cloneTaskConfig(cfg)
	return nil
}

// Update 按 id 全量更新任务，task_code 与其他记录冲突时返回 ErrDuplicate。
func (m *MemoryStore) Update(ctx context.Context, cfg *TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, t := range m.tasks {
		if t.ID != cfg.ID && t.TaskCode == cfg.TaskCode {
			return ErrDuplicate
		}
	}

	if _, ok := m.tasks[cfg.ID]; !ok {
		return ErrNotFound
	}

	m.tasks[cfg.ID] = cloneTaskConfig(cfg)
	return nil
}

// UpdateStatus 更新任务启用/禁用状态。
func (m *MemoryStore) UpdateStatus(ctx context.Context, id string, status TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	task.Status = status
	task.Version++
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

// ListEnabled 获取所有启用状态的任务。
func (m *MemoryStore) ListEnabled(ctx context.Context) ([]*TaskConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*TaskConfig
	for _, t := range m.tasks {
		if t.Status == StatusEnabled {
			result = append(result, cloneTaskConfig(t))
		}
	}
	return result, nil
}

func cloneTaskConfig(task *TaskConfig) *TaskConfig {
	cloned := *task
	return &cloned
}
