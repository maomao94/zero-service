package crontask

import (
	"context"
	"errors"
	"strings"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/gormx"

	"gorm.io/gorm"
)

var _ crontask.TaskStore = (*DBStore)(nil)

// DBStore 基于 GORM 的 TaskStore 实现，支持 MySQL/PostgreSQL/GaussDB。
// LockAndFetch 使用 SELECT + UPDATE 两段式乐观锁（参照 trigger 扫表逻辑）。
// 乐观锁由 gorm.io/plugin/optimisticlock 自动处理 WHERE version = ? 和 version 自增。
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
//  1. SELECT status=enabled AND next_run<=now，按 priority DESC + RAND() 排序，LIMIT 1
//  2. UPDATE next_run = now+lockDur WHERE next_run<=now，通过时间扩展防并发
//     RowsAffected==0 → 已被其他实例抢占，返回 ErrNotFound
func (s *DBStore) LockAndFetch(ctx context.Context, now time.Time, lockDur time.Duration) (*crontask.TaskConfig, error) {
	quietCtx := gormx.WithoutSQLTrace(ctx)

	var randomFn string
	if s.dbType == gormx.DatabasePostgres || s.dbType == gormx.DatabaseGaussDB {
		randomFn = "RANDOM()"
	} else {
		randomFn = "RAND()"
	}

	var records []gormmodel.GormTaskConfig
	err := s.db.WithContext(quietCtx).
		Where("status = ?", int(crontask.StatusEnabled)).
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

	lockedTime := now.Add(lockDur)
	result := s.db.WithContext(quietCtx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("next_run <= ?", now).
		Updates(map[string]interface{}{
			"next_run": lockedTime,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, crontask.ErrNotFound
	}

	originalNextRun := record.NextRun
	cfg := toTaskConfig(&record)
	cfg.NextRun = originalNextRun
	return cfg, nil
}

// UpdateNextRun 更新任务的下次调度时间和上次执行时间。
func (s *DBStore) UpdateNextRun(ctx context.Context, id int64, nextRun, lastRun time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"next_run": nextRun,
			"last_run": lastRun,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
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
	record := fromTaskConfig(cfg)
	result := s.db.WithContext(ctx).
		Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", cfg.ID).
		Updates(record)
	if result.Error != nil {
		if isDuplicateErr(result.Error) {
			return crontask.ErrDuplicate
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
}

// UpdateStatus 更新任务启用/禁用状态。
func (s *DBStore) UpdateStatus(ctx context.Context, id int64, status crontask.TaskStatus) error {
	record := gormmodel.GormTaskConfig{}
	record.Id = id

	result := s.db.WithContext(ctx).
		Model(&record).
		Updates(map[string]interface{}{
			"status": int(status),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
}

// Delete 软删除任务。
func (s *DBStore) Delete(ctx context.Context, id int64) error {
	result := s.db.WithContext(ctx).Where("id = ?", id).Delete(&gormmodel.GormTaskConfig{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return crontask.ErrNotFound
	}
	return nil
}

// ListEnabled 获取所有启用状态的任务配置。
func (s *DBStore) ListEnabled(ctx context.Context) ([]*crontask.TaskConfig, error) {
	var records []gormmodel.GormTaskConfig
	err := s.db.WithContext(ctx).
		Where("status = ?", int(crontask.StatusEnabled)).
		Find(&records).Error
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
