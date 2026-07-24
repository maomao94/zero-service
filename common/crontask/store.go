package crontask

import (
	"context"
	"time"
)

// TaskClaim 表示一次成功的任务抢占。
// Task.NextRun 保留原计划时间，LockedUntil 是完成更新必须携带的 lease token。
type TaskClaim struct {
	Task        *TaskConfig
	LockedUntil time.Time
}

// ListCondition 定义任务列表查询条件。
// Statuses 为空时不过滤状态；非空时返回任一指定状态的任务。
type ListCondition struct {
	Statuses []TaskStatus
}

// TaskStore 任务持久化接口，支持 DB / Redis / Memory 多种后端实现。
// task_code 全局唯一，Insert 违反时返回 ErrDuplicate。
// LockAndFetch 与 Complete 必须使用同一个 LockedUntil token 完成 lease CAS。
type TaskStore interface {
	// LockAndFetch 扫描并锁定一个到期任务。defaultLockTimeout 是调度器默认锁超时。
	// 任务配置了正数 LockTimeout 时优先使用任务值，否则使用默认值；选中后将 next_run 推迟到锁截止时间。
	LockAndFetch(ctx context.Context, now time.Time, defaultLockTimeout time.Duration) (*TaskClaim, error)

	// Complete 完成一次已抢占的周期执行。
	// expectedLockedUntil 必须匹配当前 next_run；任务执行期间被禁用不影响本次完成。
	// nextRun 零值表示无下次调度，lastRun 零值表示保留原值。
	Complete(ctx context.Context, id string, expectedLockedUntil, nextRun, lastRun time.Time) error

	// UpdateLastRun 只记录一次独立手动执行的成功时间，不修改周期计划。
	UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error

	// GetByCode 按全局唯一的 task_code 查询任务。
	GetByCode(ctx context.Context, taskCode string) (*TaskConfig, error)

	// Insert 新增任务，task_code 冲突时返回 ErrDuplicate。
	Insert(ctx context.Context, cfg *TaskConfig) error

	// Update 按 id 全量更新任务配置。
	Update(ctx context.Context, cfg *TaskConfig) error

	// Enable 启用任务，并根据已保存的 RRULE 从当前时间重新计算未来 NextRun。
	Enable(ctx context.Context, id string) error

	// Disable 禁用任务；已被 claim 的在途执行仍可按 lease token 完成本次执行。
	// 任务已处于禁用状态时幂等成功，任务不存在或已删除时返回 ErrUpdate。
	Disable(ctx context.Context, id string) error

	// Delete 删除任务（软删除）。
	Delete(ctx context.Context, id string) error

	// List 按条件获取任务配置；零值条件返回全部任务。
	List(ctx context.Context, condition ListCondition) ([]*TaskConfig, error)
}
