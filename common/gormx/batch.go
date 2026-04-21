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
			updateData := make(map[string]any, len(up)-1)
			for k, v := range up {
				if k != "id" {
					updateData[k] = v
				}
			}
			if err := tx.Model(model).Where("id = ?", id).Updates(updateData).Error; err != nil {
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
	return db.Unscoped().Model(model).Select("deleted_at").Updates(map[any]any{"deleted_at": nil}).Error
}

func UnscopedUpdate(db *gorm.DB, model any, updates map[string]any) error {
	return db.Session(&gorm.Session{
		SkipHooks: true,
	}).Model(model).Updates(updates).Error
}

func UnscopedCreate(db *gorm.DB, value any) error {
	return db.Session(&gorm.Session{
		SkipHooks: true,
	}).Create(value).Error
}

func SystemUpdate(db *gorm.DB, model any, updates map[string]any) error {
	emptyCtx := context.Background()
	return db.WithContext(emptyCtx).Model(model).Updates(updates).Error
}

func SystemCreate(db *gorm.DB, value any) error {
	emptyCtx := context.Background()
	return db.WithContext(emptyCtx).Create(value).Error
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
			updateData := make(map[string]any, len(up)-1)
			for k, v := range up {
				if k != "id" {
					updateData[k] = v
				}
			}

			query := tx.Model(model).Where("id = ?", id)
			if tenantID := GetTenantID(ctx); tenantID != "" {
				query = query.Where("tenant_id = ?", tenantID)
			}

			if err := query.Updates(updateData).Error; err != nil {
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

	query := db.WithContext(ctx).Model(model).Where("id IN ?", ids)

	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Delete(model).Error
}

func BatchDeleteByConditionWithTenant(ctx context.Context, db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := queryFn(tx)
		if tenantID := GetTenantID(ctx); tenantID != "" {
			q = q.Where("tenant_id = ?", tenantID)
		}
		return q.Delete(model).Error
	})
}

func RestoreWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	query := db.WithContext(ctx).Unscoped().Model(model).Select("deleted_at")

	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if len(conds) > 0 {
		query = query.Where(conds[0], conds[1:]...)
	}

	return query.Updates(map[any]any{"deleted_at": nil}).Error
}

func UnscopedDeleteWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	query := db.WithContext(ctx).Unscoped().Model(model)

	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if len(conds) > 0 {
		query = query.Where(conds[0], conds[1:]...)
	}

	return query.Delete(model).Error
}
