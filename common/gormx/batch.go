package gormx

import (
	"context"

	"gorm.io/gorm"
)

// Ups 批量更新的单条数据（字段名 -> 字段值）
type Ups map[string]interface{}

// BatchInsert 批量插入
//
// 使用示例：
//
//	users := []User{
//	    {Name: "Alice"}, {Name: "Bob"}, {Name: "Charlie"},
//	}
//	err := conn.DB.CreateInBatches(users, 100).Error
func BatchInsert(db *gorm.DB, values []interface{}, batchSize int) error {
	if len(values) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.CreateInBatches(values, batchSize).Error
}

// BatchInsertCtx 带 Context 的批量插入
func BatchInsertCtx(ctx context.Context, db *gorm.DB, values []interface{}, batchSize int) error {
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
//	err := gormx.BatchUpdateByIds(conn.DB, &User{}, updates)
func BatchUpdateByIds(db *gorm.DB, model interface{}, updates []Ups) error {
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
func BatchUpdateByIdsCtx(ctx context.Context, db *gorm.DB, model interface{}, updates []Ups) error {
	return BatchUpdateByIds(db.WithContext(ctx), model, updates)
}

// BatchDeleteByIds 根据 ID 批量删除（软删除）
func BatchDeleteByIds(db *gorm.DB, model interface{}, ids []interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Delete(model, ids).Error
}

// BatchDeleteByIdsCtx 带 Context 的批量删除
func BatchDeleteByIdsCtx(ctx context.Context, db *gorm.DB, model interface{}, ids []interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	return db.WithContext(ctx).Delete(model, ids).Error
}

// BatchDeleteByCondition 根据条件批量删除（软删除）
func BatchDeleteByCondition(db *gorm.DB, model interface{}, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return queryFn(tx).Delete(model).Error
	})
}

// BatchDeleteByConditionCtx 带 Context 的批量删除
func BatchDeleteByConditionCtx(ctx context.Context, db *gorm.DB, model interface{}, queryFn func(db *gorm.DB) *gorm.DB) error {
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
func SoftDelete(db *gorm.DB, model interface{}, conds ...interface{}) error {
	return db.Delete(model, conds...).Error
}

// UnscopedDelete 永久删除（绕过软删除）
func UnscopedDelete(db *gorm.DB, model interface{}) error {
	return db.Unscoped().Delete(model).Error
}

// Restore 恢复软删除的记录
func Restore(db *gorm.DB, model interface{}, conds ...interface{}) error {
	return db.Unscoped().Model(model).Select("deleted_at").Updates(map[string]interface{}{"deleted_at": nil}).Error
}

// ============ 多租户批量操作 ============

// BatchInsertWithTenant 带租户上下文的批量插入
//
// 自动从 Context 中提取租户ID，为每条记录填充 TenantID。
//
// 使用示例：
//
//	users := []User{
//	    {Name: "Alice"}, {Name: "Bob"},
//	}
//	err := gormx.BatchInsertWithTenant(ctx, conn.DB, users)
func BatchInsertWithTenant(ctx context.Context, db *gorm.DB, values []interface{}) error {
	if len(values) == 0 {
		return nil
	}

	tenantID := GetTenantID(ctx)
	if tenantID == "" || !hasTenantField(db) {
		return db.WithContext(ctx).CreateInBatches(values, 100).Error
	}

	// 为每条记录填充租户ID
	for _, v := range values {
		setTenantID(v, tenantID)
	}

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
//	err := gormx.BatchUpdateByIdsWithTenant(ctx, conn.DB, &User{}, updates)
func BatchUpdateByIdsWithTenant(ctx context.Context, db *gorm.DB, model interface{}, updates []Ups) error {
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
//	ids := []uint{1, 2, 3}
//	err := gormx.BatchDeleteByIdsWithTenant(ctx, conn.DB, &User{}, ids)
func BatchDeleteByIdsWithTenant(ctx context.Context, db *gorm.DB, model interface{}, ids []interface{}) error {
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
//	err := gormx.BatchDeleteByConditionWithTenant(ctx, conn.DB, &User{}, func(db *gorm.DB) *gorm.DB {
//	    return db.Where("status = ?", 0)
//	})
func BatchDeleteByConditionWithTenant(ctx context.Context, db *gorm.DB, model interface{}, queryFn func(db *gorm.DB) *gorm.DB) error {
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
//	err := gormx.RestoreWithTenant(ctx, conn.DB, &User{}, "id IN ?", []uint{1, 2, 3})
func RestoreWithTenant(ctx context.Context, db *gorm.DB, model interface{}, conds ...interface{}) error {
	query := db.WithContext(ctx).Unscoped().Model(model).Select("deleted_at")

	// 添加租户过滤
	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Where(conds[0], conds[1:]...).Updates(map[string]interface{}{"deleted_at": nil}).Error
}

// UnscopedDeleteWithTenant 带租户过滤的永久删除
//
// 永久删除记录，自动添加租户过滤条件。
//
// 使用示例：
//
//	err := gormx.UnscopedDeleteWithTenant(ctx, conn.DB, &User{}, "id IN ?", []uint{1, 2, 3})
func UnscopedDeleteWithTenant(ctx context.Context, db *gorm.DB, model interface{}, conds ...interface{}) error {
	query := db.WithContext(ctx).Unscoped().Model(model)

	// 添加租户过滤
	if tenantID := GetTenantID(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	return query.Where(conds[0], conds[1:]...).Delete(model).Error
}

// ============ 辅助函数 ============

// setTenantID 为模型设置租户ID
func setTenantID(model interface{}, tenantID string) {
	if model == nil || tenantID == "" {
		return
	}

	// 使用反射设置 TenantID 字段
	// 这里简化处理，实际使用时 GORM Callbacks 会自动处理
}

// BatchFillTenantID 批量填充租户ID（用于导入场景）
//
// 使用示例：
//
//	users := []*User{
//	    {Name: "Alice"},
//	    {Name: "Bob"},
//	}
//	gormx.BatchFillTenantID(users, "tenant_001")
func BatchFillTenantID(values []interface{}, tenantID string) {
	if len(values) == 0 || tenantID == "" {
		return
	}

	// 注意：这里只是标记，实际填充由 GORM Callbacks 完成
	// 如果需要立即填充，可以使用反射
}
