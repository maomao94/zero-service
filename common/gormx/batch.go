package gormx

import (
	"context"

	"gorm.io/gorm"
)

type Ups map[string]any

func BatchInsert[T any](db *gorm.DB, values []T, batchSize int) error {
	if len(values) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.CreateInBatches(values, batchSize).Error
}

func BatchUpdateByIds(db *gorm.DB, model any, updates []Ups) error {
	if len(updates) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, up := range updates {
			id, ok := up["id"]
			if !ok {
				continue
			}
			data := make(map[string]any, len(up)-1)
			for k, v := range up {
				if k != "id" {
					data[k] = v
				}
			}
			if err := tx.Model(model).Where("id = ?", id).Updates(data).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchDeleteByIds[T any](db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Delete(model, ids).Error
}

func BatchDeleteByCondition(db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return queryFn(tx).Delete(model).Error
	})
}

func SoftDelete(db *gorm.DB, model any, conds ...any) error {
	return db.Delete(model, conds...).Error
}

func UnscopedDelete(db *gorm.DB, model any) error {
	return db.Unscoped().Delete(model).Error
}

func Restore(db *gorm.DB, model any, conds ...any) error {
	q := db.Unscoped().Model(model)
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if hasLegacyDeleteFields(q) {
		return q.Select("delete_time", "del_state").Updates(map[string]any{
			"delete_time": nil,
			"del_state":   int64(0),
		}).Error
	}
	return q.Update("deleted_at", nil).Error
}

func hasLegacyDeleteFields(db *gorm.DB) bool {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(db.Statement.Model); err != nil {
		return false
	}
	_, hasDeleteTime := stmt.Schema.FieldsByDBName["delete_time"]
	_, hasDelState := stmt.Schema.FieldsByDBName["del_state"]
	return hasDeleteTime && hasDelState
}

func UnscopedUpdate(db *gorm.DB, model any, updates map[string]any) error {
	return db.Session(&gorm.Session{SkipHooks: true}).Model(model).Updates(updates).Error
}

func UnscopedCreate(db *gorm.DB, value any) error {
	return db.Session(&gorm.Session{SkipHooks: true}).Create(value).Error
}

func withTenantQuery(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tenantID := GetTenantID(ctx); tenantID != "" {
		return db.Where("tenant_id = ?", tenantID)
	}
	return db
}

func BatchInsertWithTenant[T any](ctx context.Context, db *gorm.DB, values []T) error {
	if len(values) == 0 {
		return nil
	}
	return db.WithContext(ctx).CreateInBatches(values, 100).Error
}

func BatchUpdateByIdsWithTenant(ctx context.Context, db *gorm.DB, model any, updates []Ups) error {
	if len(updates) == 0 {
		return nil
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, up := range updates {
			id, ok := up["id"]
			if !ok {
				continue
			}
			data := make(map[string]any, len(up)-1)
			for k, v := range up {
				if k != "id" {
					data[k] = v
				}
			}
			q := withTenantQuery(ctx, tx.Model(model).Where("id = ?", id))
			if err := q.Updates(data).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchDeleteByIdsWithTenant[T any](ctx context.Context, db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	q := withTenantQuery(ctx, db.WithContext(ctx).Model(model).Where("id IN ?", ids))
	return q.Delete(model).Error
}

func BatchDeleteByConditionWithTenant(ctx context.Context, db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return withTenantQuery(ctx, queryFn(tx)).Delete(model).Error
	})
}

func RestoreWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	q := withTenantQuery(ctx, db.WithContext(ctx).Unscoped().Model(model))
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if hasLegacyDeleteFields(q) {
		return q.Select("delete_time", "del_state").Updates(map[string]any{
			"delete_time": nil,
			"del_state":   int64(0),
		}).Error
	}
	return q.Update("deleted_at", nil).Error
}

func UnscopedDeleteWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	q := withTenantQuery(ctx, db.WithContext(ctx).Unscoped().Model(model))
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	return q.Delete(model).Error
}
