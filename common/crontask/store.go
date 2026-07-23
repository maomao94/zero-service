package crontask

import (
	"context"
	"time"
)

// TaskStore 任务持久化接口，支持 DB / Redis / Memory 多种后端实现。
// task_code 全局唯一，Insert 违反时返回 ErrDuplicate。
// LockAndFetch 需实现乐观锁时间扩展，防止多实例重复调度。
type TaskStore interface {
	// LockAndFetch 扫描并锁定一个到期任务。now 为当前时间，lockDur 为锁定窗口。
	// 选中任务后应将 next_run 推迟 now+lockDur，防止被其他实例重复扫到。
	LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*TaskConfig, error)

	// UpdateNextRun 更新任务的下次调度时间和上次执行时间，nextRun 零值表示无下次调度。
	UpdateNextRun(ctx context.Context, id string, nextRun, lastRun time.Time) error

	// GetByCode 按全局唯一的 task_code 查询任务。
	GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error)

	// Insert 新增任务，task_code 冲突时返回 ErrDuplicate。
	Insert(ctx context.Context, cfg *TaskConfig) error

	// Update 按 id 全量更新任务配置。
	Update(ctx context.Context, cfg *TaskConfig) error

	// UpdateStatus 更新任务启用/禁用状态。
	UpdateStatus(ctx context.Context, id string, status TaskStatus) error

	// Delete 删除任务（软删除）。
	Delete(ctx context.Context, id string) error

	// ListEnabled 获取所有启用状态的任务配置。
	ListEnabled(ctx context.Context) ([]*TaskConfig, error)
}
