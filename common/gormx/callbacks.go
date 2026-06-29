package gormx

import (
	"gorm.io/gorm"
)

func RegisterCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("gormx:before_create", beforeCreateHook)
	db.Callback().Update().Before("gorm:update").Register("gormx:before_update", beforeUpdateHook)
	db.Callback().Delete().Before("gorm:delete").Register("gormx:before_delete", beforeDeleteHook)
}

func beforeCreateHook(db *gorm.DB) {
	userCtx := GetUserContext(db.Statement.Context)
	if userCtx == nil {
		return
	}

	if userID := userCtx.AuditUserValue(); userID != nil {
		setSchemaColumn(db, "create_user", userID)
		setSchemaColumn(db, "create_name", userCtx.UserName)
		setSchemaColumn(db, "update_user", userID)
		setSchemaColumn(db, "update_name", userCtx.UserName)
	}

	if userCtx.TenantID != "" && HasTenantField(db) {
		setSchemaColumn(db, "tenant_id", userCtx.TenantID)
	}
}

func beforeUpdateHook(db *gorm.DB) {
	userCtx := GetUserContext(db.Statement.Context)
	if userCtx == nil {
		return
	}
	if userID := userCtx.AuditUserValue(); userID != nil {
		setSchemaColumn(db, "update_user", userID)
		setSchemaColumn(db, "update_name", userCtx.UserName)
	}
	// version 由 gorm.io/plugin/optimisticlock 的 Version 类型自动处理
}

func beforeDeleteHook(db *gorm.DB) {
	userCtx := GetUserContext(db.Statement.Context)
	if userCtx == nil {
		return
	}
	if userID := userCtx.AuditUserValue(); userID != nil {
		setSchemaColumn(db, "delete_user", userID)
		setSchemaColumn(db, "delete_name", userCtx.UserName)
	}
}

func setSchemaColumn(db *gorm.DB, column string, value any) {
	if db.Statement.Schema == nil {
		return
	}
	if _, ok := db.Statement.Schema.FieldsByDBName[column]; !ok {
		return
	}
	db.Statement.SetColumn(column, value)
}
