package gormx

import (
	"context"

	"gorm.io/gorm"
)

// TenantScope 返回租户过滤 Scope
//
// 自动从 Context 中获取租户ID，添加 WHERE tenant_id = ? 条件。
// 如果 Context 中没有租户信息，则返回原查询不过滤。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantScope(ctx)).Find(&configs).Error
//
//	// 相当于
//	err := db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&configs).Error
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// 超级管理员不过滤租户
		if IsSuperAdmin(ctx) {
			return db
		}

		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			return db // 非多租户场景，返回原查询
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db
		}

		return db.Where("tenant_id = ?", userCtx.TenantID)
	}
}

// TenantScopeStrict 严格模式租户过滤
//
// 租户ID 必须存在，否则返回空结果集。
// 适用于必须指定租户的业务场景。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantScopeStrict(ctx)).Find(&configs).Error
//
//	// 如果没有租户上下文，相当于 WHERE 1=0
func TenantScopeStrict(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// 超级管理员不过滤租户
		if IsSuperAdmin(ctx) {
			return db
		}

		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			// 强制返回空结果
			return db.Where("1 = 0")
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db
		}

		return db.Where("tenant_id = ?", userCtx.TenantID)
	}
}

// TenantScopeWithDelete 租户过滤（包含已删除记录）
//
// 在某些查询场景下需要查看已软删除的记录。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(
//	    gormx.TenantScopeWithDelete(ctx),
//	    func(db *gorm.DB) *gorm.DB { return db.Unscoped() },
//	).Find(&configs).Error
func TenantScopeWithDelete(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// 超级管理员不过滤租户
		if IsSuperAdmin(ctx) {
			return db.Unscoped()
		}

		userCtx := GetUserContext(ctx)
		if userCtx == nil || userCtx.TenantID == "" {
			return db.Unscoped()
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db.Unscoped()
		}

		return db.Unscoped().Where("tenant_id = ?", userCtx.TenantID)
	}
}

// TenantEq 直接指定租户ID 的 Scope
//
// 用于需要明确指定租户的场景。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantEq("tenant_001")).Find(&configs).Error
func TenantEq(tenantID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantID == "" {
			return db
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db
		}

		return db.Where("tenant_id = ?", tenantID)
	}
}

// TenantScopeWithSuperAdmin 超级管理员模式的租户过滤
//
// 超级管理员可访问所有租户数据，普通用户按租户过滤。
// 这是最常用的租户过滤模式。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantScopeWithSuperAdmin(ctx)).Find(&configs).Error
func TenantScopeWithSuperAdmin(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return TenantScope(ctx) // TenantScope 已经内置了超级管理员判断
}

// TenantNotEq 非指定租户的 Scope
//
// 用于查询不属于某个租户的数据。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantNotEq("tenant_001")).Find(&configs).Error
func TenantNotEq(tenantID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantID == "" {
			return db
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db
		}

		return db.Where("tenant_id != ?", tenantID)
	}
}

// TenantIn 多租户查询 Scope
//
// 用于查询多个租户的数据（如数据导出等场景）。
//
// 使用示例：
//
//	var configs []*UserConfig
//	err := db.WithContext(ctx).Scopes(gormx.TenantIn("tenant_001", "tenant_002")).Find(&configs).Error
func TenantIn(tenantIDs ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(tenantIDs) == 0 {
			return db
		}

		// 检查模型是否有租户字段
		if !hasTenantField(db) {
			return db
		}

		return db.Where("tenant_id IN ?", tenantIDs)
	}
}

// WithTenantContext 创建带租户上下文的 Context
//
// 便捷函数，用于快速创建带租户信息的 Context。
//
// 使用示例：
//
//	ctx := gormx.WithTenantContext(context.Background(), "tenant_001")
func WithTenantContext(ctx context.Context, tenantID string) context.Context {
	return WithUserContext(ctx, &UserContext{
		TenantID: tenantID,
	})
}

// WithUserAndTenantContext 创建带用户和租户上下文的 Context
//
// 使用示例：
//
//	ctx := gormx.WithUserAndTenantContext(ctx, 1, "admin", "tenant_001")
func WithUserAndTenantContext(ctx context.Context, userID uint, userName, tenantID string) context.Context {
	return WithUserContext(ctx, &UserContext{
		UserID:   userID,
		UserName: userName,
		TenantID: tenantID,
	})
}

// HasTenantField 检查模型是否有租户字段
// 导出的版本，供外部使用
func HasTenantField(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]
	return ok
}
