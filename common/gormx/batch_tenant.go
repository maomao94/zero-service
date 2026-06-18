package gormx

import "gorm.io/gorm"

func BatchInsertWithTenant[T any](db *gorm.DB, values []T) error {
	if len(values) == 0 {
		return nil
	}
	return db.CreateInBatches(values, 100).Error
}

func BatchUpdateByIdsWithTenant(db *gorm.DB, model any, updates []Ups) error {
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
			q := withTenantQueryFromDB(tx.Model(model).Where("id = ?", id))
			if err := q.Updates(data).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchDeleteByIdsWithTenant[T any](db *gorm.DB, model *T, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	q := withTenantQueryFromDB(db.Model(model).Where("id IN ?", ids))
	return q.Delete(model).Error
}

func BatchDeleteByConditionWithTenant(db *gorm.DB, model any, queryFn func(db *gorm.DB) *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return withTenantQueryFromDB(queryFn(tx)).Delete(model).Error
	})
}
