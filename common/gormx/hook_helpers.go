package gormx

import "gorm.io/gorm"

func SkipHooksUpdate(db *gorm.DB, model any, updates map[string]any) error {
	return db.Session(&gorm.Session{SkipHooks: true}).Model(model).Updates(updates).Error
}

func SkipHooksCreate(db *gorm.DB, value any) error {
	return db.Session(&gorm.Session{SkipHooks: true}).Create(value).Error
}
