package gormx

import (
	"context"

	"gorm.io/gorm"
)

func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			return db
		}
		if !HasTenantField(db) {
			return db
		}
		return db.Where("tenant_id = ?", userCtx.TenantID)
	}
}

func TenantScopeStrict(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			return db.Where("1 = 0")
		}
		if !HasTenantField(db) {
			return db
		}
		return db.Where("tenant_id = ?", userCtx.TenantID)
	}
}

func TenantScopeWithDelete(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			return db.Unscoped()
		}
		if !HasTenantField(db) {
			return db.Unscoped()
		}
		return db.Unscoped().Where("tenant_id = ?", userCtx.TenantID)
	}
}

func TenantEq(tenantID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantID == "" || !HasTenantField(db) {
			return db
		}
		return db.Where("tenant_id = ?", tenantID)
	}
}

func TenantNotEq(tenantID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantID == "" || !HasTenantField(db) {
			return db
		}
		return db.Where("tenant_id != ?", tenantID)
	}
}

func TenantIn(tenantIDs ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(tenantIDs) == 0 || !HasTenantField(db) {
			return db
		}
		return db.Where("tenant_id IN ?", tenantIDs)
	}
}

func WithTenantContext(ctx context.Context, tenantID string) context.Context {
	return WithUserContext(ctx, &UserContext{TenantID: tenantID})
}

func WithUserAndTenantContext[T AuditUserID](ctx context.Context, userID T, userName, tenantID string) context.Context {
	return WithUserContext(ctx, NewUserContext(userID, userName, tenantID))
}

func WithStringUserAndTenantContext(ctx context.Context, userID, userName, tenantID string) context.Context {
	return WithUserAndTenantContext(ctx, userID, userName, tenantID)
}

func HasTenantField(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]
	return ok
}
