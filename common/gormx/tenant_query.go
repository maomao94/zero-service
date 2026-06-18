package gormx

import "gorm.io/gorm"

func withTenantQueryFromDB(db *gorm.DB) *gorm.DB {
	if tenantID := GetTenantID(db.Statement.Context); tenantID != "" {
		return db.Where("tenant_id = ?", tenantID)
	}
	return db
}
