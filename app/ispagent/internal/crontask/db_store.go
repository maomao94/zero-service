package crontask

import (
	"context"
	"errors"
	"strings"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/rrulex"

	"gorm.io/gorm"
)

var _ crontask.TaskStore = (*DBStore)(nil)

// DBStore 基于 GORM 的 TaskStore 实现，支持 MySQL/PostgreSQL/GaussDB。
// LockAndFetch 和 Complete 使用 next_run lease token 防止并发覆盖。
type DBStore struct {
	db     *gormx.DB
	dbType gormx.DatabaseType
}

func NewDBStore(db *gormx.DB) *DBStore {
	return &DBStore{
		db:     db,
		dbType: gormx.GetDatabaseTypeFromDialector(db.DB),
	}
}

// LockAndFetch 扫描并锁定一个到期任务，参照 trigger 的 LockTriggerItem 模式：
//  1. SELECT status=enabled AND next_run<=now，按 priority DESC + 随机函数排序，LIMIT 1
//  2. UPDATE next_run = now+lockDur WHERE next_run<=now，通过时间扩展防并发
//     RowsAffected==0 → 已被其他实例抢占，返回 ErrNotFound
func (s *DBStore) LockAndFetch(ctx context.Context, now time.Time, defaultLockTimeout time.Duration) (*crontask.TaskClaim, error) {
	quietCtx := gormx.WithoutSQLTrace(ctx)

	var randomFn string
	if s.dbType == gormx.DatabasePostgres || s.dbType == gormx.DatabaseSQLite {
		randomFn = "RANDOM()"
	} else {
		randomFn = "RAND()"
	}

	var records []gormmodel.GormTaskConfig
	err := s.db.WithContext(quietCtx).
		Where("status = ?", int(crontask.StatusEnabled)).
		Where("next_run IS NOT NULL").
		Where("next_run <= ?", now).
		Order("priority DESC, " + randomFn).
		Limit(1).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, crontask.ErrNotFound
	}
	record := records[0]

	lockTimeout := crontask.ResolveLockTimeout(time.Duration(record.LockTimeout)*time.Millisecond, defaultLockTimeout)
	lockedTime := now.Add(lockTimeout).Truncate(time.Second)
	scheduledTime := record.NextRun.Time
	if record.ScheduledTime.Valid {
		scheduledTime = record.ScheduledTime.Time
	}
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", record.Id).
		Where("status = ?", int(crontask.StatusEnabled)).
		Where("next_run = ?", record.NextRun.Time).
		Updates(map[string]interface{}{
			"next_run":       lockedTime,
			"scheduled_time": scheduledTime,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, crontask.ErrNotFound
	}

	task := toTaskConfig(&record)
	task.ScheduledTime = scheduledTime
	task.NextRun = time.Time{}
	return &crontask.TaskClaim{Task: task, LockedUntil: lockedTime}, nil
}

// Complete 使用 LockedUntil token 完成一次周期执行。
func (s *DBStore) Complete(ctx context.Context, id string, expectedLockedUntil time.Time, completion crontask.Completion) error {
	updates := map[string]interface{}{
		"next_run":       carbonx.ToNullTime(completion.NextRun),
		"scheduled_time": nil,
	}
	if !completion.LastRun.IsZero() {
		updates["last_run"] = completion.LastRun
	}
	if !completion.LastScheduledRun.IsZero() {
		updates["last_scheduled_run"] = completion.LastScheduledRun
	}
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", id).
		Where("next_run = ?", expectedLockedUntil).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
}

// UpdateLastRun 只记录一次独立手动执行的成功时间。
func (s *DBStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", id).
		Update("last_run", lastRun)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
}

// GetByID 按任务 ID 查询任务配置。
func (s *DBStore) GetByID(ctx context.Context, id string) (*crontask.TaskConfig, error) {
	var record gormmodel.GormTaskConfig
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, crontask.ErrNotFound
		}
		return nil, err
	}
	return toTaskConfig(&record), nil
}

// GetByCode 按全局唯一的 task_code 查询任务配置。
func (s *DBStore) GetByCode(ctx context.Context, taskCode string) (*crontask.TaskConfig, error) {
	var record gormmodel.GormTaskConfig
	err := s.db.WithContext(ctx).Where("task_code = ?", taskCode).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, crontask.ErrNotFound
		}
		return nil, err
	}
	return toTaskConfig(&record), nil
}

// Insert 新增任务配置。task_code 违反唯一约束时返回 ErrDuplicate。
func (s *DBStore) Insert(ctx context.Context, cfg *crontask.TaskConfig) error {
	if err := rrulex.Validate(cfg.RRuleStr); err != nil {
		return err
	}
	record := fromTaskConfig(cfg)
	err := s.db.WithContext(ctx).Create(record).Error
	if err != nil {
		if isDuplicateErr(err) {
			return crontask.ErrDuplicate
		}
		return err
	}
	return nil
}

// Update 按 id 全量更新任务配置。task_code 违反唯一约束时返回 ErrDuplicate。
func (s *DBStore) Update(ctx context.Context, cfg *crontask.TaskConfig) error {
	if err := rrulex.Validate(cfg.RRuleStr); err != nil {
		return err
	}
	record := fromTaskConfig(cfg)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&gormmodel.GormTaskConfig{}).
			Where("id = ?", cfg.ID).
			Select("*").
			Omit("id", "create_time", "delete_time", "is_deleted", "last_run", "last_scheduled_run", "scheduled_time", "next_run").
			Updates(record)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return crontask.ErrNotFound
		}
		return tx.Model(&gormmodel.GormTaskConfig{}).
			Where("id = ?", cfg.ID).
			Where("scheduled_time IS NULL").
			Update("next_run", carbonx.ToNullTime(cfg.NextRun)).Error
	})
	if err != nil {
		if isDuplicateErr(err) {
			return crontask.ErrDuplicate
		}
		return err
	}
	return nil
}

// Enable 启用任务，并根据已保存的 RRULE 从当前时间重新计算未来 NextRun。
func (s *DBStore) Enable(ctx context.Context, id string) error {
	var record gormmodel.GormTaskConfig
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return crontask.ErrNotFound
		}
		return err
	}
	if crontask.TaskStatus(record.Status) == crontask.StatusEnabled {
		return nil
	}
	nextRun := record.NextRun.Time
	if record.ScheduledTime.Valid {
		nextRun = record.ScheduledTime.Time
	}
	if record.RRuleStr != "" {
		set, err := rrulex.ParseSet(record.RRuleStr)
		if err != nil {
			return err
		}
		nextRun = set.After(time.Now(), false)
	}
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         int(crontask.StatusEnabled),
			"next_run":       carbonx.ToNullTime(nextRun),
			"scheduled_time": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Disable 禁用任务，不撤销已经 claim 的在途执行。
func (s *DBStore) Disable(ctx context.Context, id string) error {
	var record gormmodel.GormTaskConfig
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return crontask.ErrUpdate
	}
	if crontask.TaskStatus(record.Status) == crontask.StatusDisabled {
		return nil
	}
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", id).
		Update("status", int(crontask.StatusDisabled))
	if result.Error != nil || result.RowsAffected == 0 {
		return crontask.ErrUpdate
	}
	return nil
}

// Delete 幂等软删除任务。
func (s *DBStore) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Where("id = ?", id).Delete(&gormmodel.GormTaskConfig{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// List 按条件获取任务配置；零值条件返回全部任务。
func (s *DBStore) List(ctx context.Context, condition crontask.ListCondition) ([]*crontask.TaskConfig, error) {
	var records []gormmodel.GormTaskConfig
	query := s.db.DB.WithContext(ctx)
	if len(condition.Statuses) > 0 {
		query = query.Where("status IN ?", condition.Statuses)
	}
	err := query.Find(&records).Error
	if err != nil {
		return nil, err
	}
	result := make([]*crontask.TaskConfig, 0, len(records))
	for i := range records {
		result = append(result, toTaskConfig(&records[i]))
	}
	return result, nil
}

// isDuplicateErr 判断是否为数据库唯一约束冲突错误。
func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "UNIQUE constraint")
}
