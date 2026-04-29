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
	if userCtx != nil {
		if userID := userCtx.AuditUserValue(); userID != nil {
			setSchemaColumn(db, "update_user", userID)
			setSchemaColumn(db, "update_name", userCtx.UserName)
		}
	}

	if hasSchemaField(db, "version") {
		db.Statement.SetColumn("version", gorm.Expr("version + 1"))
	}
}

func beforeDeleteHook(db *gorm.DB) {
	userCtx := GetUserContext(db.Statement.Context)
	if userCtx != nil {
		if userID := userCtx.AuditUserValue(); userID != nil {
			setSchemaColumn(db, "delete_user", userID)
			setSchemaColumn(db, "delete_name", userCtx.UserName)
		}
	}
}

func hasSchemaField(db *gorm.DB, field string) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName[field]
	return ok
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
