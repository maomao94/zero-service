package gormx

import "gorm.io/gorm"

func RegisterCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("gormx:before_create", beforeCreateHook)
	db.Callback().Update().Before("gorm:update").Register("gormx:before_update", beforeUpdateHook)
	db.Callback().Delete().Before("gorm:delete").Register("gormx:before_delete", beforeDeleteHook)
}

func beforeCreateHook(db *gorm.DB) {}

func beforeUpdateHook(db *gorm.DB) {}

func beforeDeleteHook(db *gorm.DB) {}
