package gormx

import (
	"context"

	"gorm.io/gorm"
)

func RegisterCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("gormx:before_create", BeforeCreateHook)
	db.Callback().Update().Before("gorm:update").Register("gormx:before_update", BeforeUpdateHook)
	db.Callback().Delete().Before("gorm:delete").Register("gormx:before_delete", BeforeDeleteHook)
}

func BeforeCreateHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := GetUserContext(ctx)

	if userCtx == nil {
		return
	}

	if userCtx.UserID > 0 {
		setColumnIfZero(db, "create_user", userCtx.UserID)
		setColumnIfZero(db, "create_name", userCtx.UserName)
		setColumnIfZero(db, "update_user", userCtx.UserID)
		setColumnIfZero(db, "update_name", userCtx.UserName)
	}

	if userCtx.TenantID != "" && HasTenantField(db) {
		setColumnIfZero(db, "tenant_id", userCtx.TenantID)
	}
}

func BeforeUpdateHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := GetUserContext(ctx)

	if userCtx != nil && userCtx.UserID > 0 {
		setColumnIfZero(db, "update_user", userCtx.UserID)
		setColumnIfZero(db, "update_name", userCtx.UserName)
	}

	if hasVersionField(db) {
		db.Statement.SetColumn("version", gorm.Expr("version + 1"))
	}
}

func BeforeDeleteHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := GetUserContext(ctx)

	if userCtx != nil && userCtx.UserID > 0 {
		setColumnIfZero(db, "delete_user", userCtx.UserID)
		setColumnIfZero(db, "delete_name", userCtx.UserName)
	}
}

func hasVersionField(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["version"]
	return ok
}

func setColumnIfZero(db *gorm.DB, column string, value any) {
	if db.Statement.Schema == nil {
		return
	}
	if _, ok := db.Statement.Schema.FieldsByDBName[column]; !ok {
		return
	}
	db.Statement.SetColumn(column, value)
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
