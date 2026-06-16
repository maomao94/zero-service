package gormx

import "gorm.io/gorm"

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
