package gormx

import (
	"context"
)

// UserContextKey 用户上下文存储的 key
const UserContextKey = "user_context"

// SuperAdminKey 超级管理员标记的 key
type SuperAdminKey struct{}

// UserContext 用户上下文
// 用于在 Context 中传递当前用户和租户信息
type UserContext struct {
	UserID   uint   // 用户ID
	UserName string // 用户姓名
	TenantID string // 租户ID（多租户场景使用）
}

// WithUserContext 向 Context 注入用户上下文
//
// 使用示例:
//
//	ctx := gormx.WithUserContext(ctx, &gormx.UserContext{
//	    UserID:   1,
//	    UserName: "admin",
//	    TenantID: "tenant_001",
//	})
func WithUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, UserContextKey, userCtx)
}

// GetUserContext 从 Context 获取用户上下文
//
// 使用示例:
//
//	userCtx := gormx.GetUserContext(ctx)
//	if userCtx != nil {
//	    fmt.Printf("UserID: %d, TenantID: %s\n", userCtx.UserID, userCtx.TenantID)
//	}
func GetUserContext(ctx context.Context) *UserContext {
	if ctx == nil {
		return nil
	}
	userCtx, ok := ctx.Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}

// GetTenantID 从 Context 获取租户ID
func GetTenantID(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	return userCtx.TenantID
}

// GetUserID 从 Context 获取用户ID
func GetUserID(ctx context.Context) uint {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return 0
	}
	return userCtx.UserID
}

// GetUserName 从 Context 获取用户名
func GetUserName(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	return userCtx.UserName
}

// WithSuperAdmin 向 Context 标记超级管理员
// 超级管理员可访问所有租户的数据
//
// 使用示例:
//
//	ctx = gormx.WithSuperAdmin(ctx)
func WithSuperAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, SuperAdminKey{}, struct{}{})
}

// IsSuperAdmin 检查 Context 是否为超级管理员
// 超级管理员不受租户数据隔离限制
//
// 使用示例:
//
//	if gormx.IsSuperAdmin(ctx) {
//	    // 执行跨租户操作
//	}
func IsSuperAdmin(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(SuperAdminKey{}).(struct{})
	return ok
}

// NewUserContext 创建用户上下文
//
// 使用示例:
//
//	userCtx := gormx.NewUserContext(1, "admin", "tenant_001")
func NewUserContext(userID uint, userName, tenantID string) *UserContext {
	return &UserContext{
		UserID:   userID,
		UserName: userName,
		TenantID: tenantID,
	}
}

// HasTenant 检查用户上下文是否有租户ID
func (u *UserContext) HasTenant() bool {
	return u != nil && u.TenantID != ""
}

// Clone 创建用户上下文的副本
func (u *UserContext) Clone() *UserContext {
	if u == nil {
		return nil
	}
	return &UserContext{
		UserID:   u.UserID,
		UserName: u.UserName,
		TenantID: u.TenantID,
	}
}
