package gormx

import (
	"context"

	"gorm.io/gorm"
)

// Ups 批量更新的单条数据（字段名 -> 字段值）
type Ups map[string]any

// BatchInsert 批量插入
//
// 使用示例：
//
//	users := []User{
//	    {Name: "Alice"}, {Name: "Bob"}, {Name: "Charlie"},
//	}
//	err := gormx.BatchInsert(db, users, 100)
func BatchInsert[T any](db *gorm.DB, values []T, batchSize int) error {
	if len(values) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.CreateInBatches(values, batchSize).Error
}

// BatchInsertCtx 带 Context 的批量插入
func BatchInsertCtx[T any](ctx context.Context, db *gorm.DB, values []T, batchSize int) error {
	if len(values) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.WithContext(ctx).CreateInBatches(values, batchSize).Error
}

// BatchUpdateByIds 根据 ID 批量更新
//
// 使用示例：
//
//	updates := []gormx.Ups{
//	    {"id": uint(1), "name": "Alice"},
//	    {"id": uint(2), "name": "Bob"},
//	}
//	err := gormx.BatchUpdateByIds(db, &User{}, updates)
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
			delete(up, "id")
			if err := tx.Model(model).Where("id = ?", id).Updates(up).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchUpdateByIdsCtx 带 Context 的批量更新
func BatchUpdateByIdsCtx(ctx context.Context, db *gorm.DB, model any, updates []Ups) error {
	return BatchUpdateByIds(db.WithContext(ctx), model, updates)
}

// BatchDeleteByIds 根据 ID 批量删除（软删除）
func BatchDeleteByIds[T any](db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Delete(model, ids).Error
}

// BatchDeleteByIdsCtx 带 Context 的批量删除
func BatchDeleteByIdsCtx[T any](ctx context.Context, db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.WithContext(ctx).Delete(model, ids).Error
}

// BatchDeleteByCondition 根据条件批量删除（软删除）
func BatchDeleteByCondition(db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return queryFn(tx).Delete(model).Error
	})
}

// BatchDeleteByConditionCtx 带 Context 的批量删除
func BatchDeleteByConditionCtx(ctx context.Context, db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return BatchDeleteByCondition(db.WithContext(ctx), model, queryFn)
}

// SoftDelete 软删除（GORM 原生支持）
//
// 说明：直接使用 GORM 的 Delete 方法即可，GORM 会自动处理软删除
//
//	var user User
//	db.Delete(&user)  // UPDATE users SET deleted_at = NOW() WHERE ...
//
//	// 根据条件软删除
//	db.Delete(&User{}, "status = ?", 0)
func SoftDelete(db *gorm.DB, model any, conds ...any) error {
	return db.Delete(model, conds...).Error
}

// UnscopedDelete 永久删除（绕过软删除）
func UnscopedDelete(db *gorm.DB, model any) error {
	return db.Unscoped().Delete(model).Error
}

// Restore 恢复软删除的记录
func Restore(db *gorm.DB, model any, conds ...any) error {
	return db.Unscoped().Model(model).Select("deleted_at").Updates(map[any]any{"deleted_at": nil}).Error
}

// ============ 多租户批量操作 ============

// BatchInsertWithTenant 带租户上下文的批量插入
//
// 自动从 Context 中提取租户ID，GORM Callbacks 会自动填充 TenantID。
//
// 使用示例：
//
//	users := []User{
//	    {Name: "Alice"}, {Name: "Bob"},
//	}
//	err := gormx.BatchInsertWithTenant(ctx, db, users)
func BatchInsertWithTenant[T any](ctx context.Context, db *gorm.DB, values []T) error {
	if len(values) == 0 {
		return nil
	}

	tenantID := GetTenantID(ctx)
	if tenantID == "" || !HasTenantField(db) {
		return db.WithContext(ctx).CreateInBatches(values, 100).Error
	}

	// GORM Callbacks 会自动填充租户ID，这里直接插入
	return db.WithContext(ctx).CreateInBatches(values, 100).Error
}

// BatchUpdateByIdsWithTenant 带租户过滤的批量更新
//
// 根据 ID 批量更新，自动添加租户过滤条件。
//
// 使用示例：
//
//	updates := []gormx.Ups{
//	    {"id": uint(1), "name": "Alice"},
//	    {"id": uint(2), "name": "Bob"},
//	}
//	err := gormx.BatchUpdateByIdsWithTenant(ctx, db, &User{}, updates)
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
			delete(up, "id")

			// 添加租户过滤
			query := tx.Model(model).Where("id = ?", id)
			if tenantID := GetTenantID(ctx); tenantID != "" {
				query = query.Where("tenant_id = ?", tenantID)
			}

			if err := query.Updates(up).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchDeleteByIdsWithTenant 带租户过滤的批量删除（软删除）
//
// 根据 ID 批量删除，自动添加租户过滤条件，自动填充删除人信息。
//
// 使用示例：
//
//	ids := []int64{1, 2, 3}
//	err := gormx.BatchDeleteByIdsWithTenant(ctx, db, &User{}, ids)
func BatchDeleteByIdsWithTenant[T any](ctx context.Context, db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := db.WithContext(ctx).Model(model).Where("id IN ?", ids)

	// 添加租户过滤
	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Delete(model).Error
}

// BatchDeleteByConditionWithTenant 带租户过滤的条件批量删除
//
// 根据自定义条件批量删除，自动添加租户过滤条件。
//
// 使用示例：
//
//	err := gormx.BatchDeleteByConditionWithTenant(ctx, db, &User{}, func(db *gorm.DB) *gorm.DB {
//	    return db.Where("status = ?", 0)
//	})
func BatchDeleteByConditionWithTenant(ctx context.Context, db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := queryFn(tx)
		// 添加租户过滤
		if tenantID := GetTenantID(ctx); tenantID != "" {
			q = q.Where("tenant_id = ?", tenantID)
		}
		return q.Delete(model).Error
	})
}

// RestoreWithTenant 带租户过滤的恢复软删除
//
// 恢复已软删除的记录，自动添加租户过滤条件。
//
// 使用示例：
//
//	err := gormx.RestoreWithTenant(ctx, db, &User{}, "id IN ?", []int64{1, 2, 3})
func RestoreWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	query := db.WithContext(ctx).Unscoped().Model(model).Select("deleted_at")

	// 添加租户过滤
	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Where(conds[0], conds[1:]...).Updates(map[any]any{"deleted_at": nil}).Error
}

// UnscopedDeleteWithTenant 带租户过滤的永久删除
//
// 永久删除记录，自动添加租户过滤条件。
//
// 使用示例：
//
//	err := gormx.UnscopedDeleteWithTenant(ctx, db, &User{}, "id IN ?", []int64{1, 2, 3})
func UnscopedDeleteWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	query := db.WithContext(ctx).Unscoped().Model(model)

	// 添加租户过滤
	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Where(conds[0], conds[1:]...).Delete(model).Error
}
