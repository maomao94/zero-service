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
