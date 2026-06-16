package gormx

import (
	"context"

	"gorm.io/gorm"
)

func withTenantQuery(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tenantID := GetTenantID(ctx); tenantID != "" {
		return db.Where("tenant_id = ?", tenantID)
	}
	return db
}
