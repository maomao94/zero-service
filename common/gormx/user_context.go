package gormx

import (
	"context"
)

type userContextKey struct{}

type superAdminKey struct{}

type UserContext struct {
	UserID   uint   `json:"user_id"`
	UserName string `json:"user_name"`
	TenantID string `json:"tenant_id"`
}

func WithUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey{}, userCtx)
}

func GetUserContext(ctx context.Context) *UserContext {
	if ctx == nil {
		return nil
	}
	userCtx, ok := ctx.Value(userContextKey{}).(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}

func GetTenantID(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	return userCtx.TenantID
}

func GetUserID(ctx context.Context) uint {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return 0
	}
	return userCtx.UserID
}

func GetUserName(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	return userCtx.UserName
}

func WithSuperAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, superAdminKey{}, struct{}{})
}

func IsSuperAdmin(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(superAdminKey{}).(struct{})
	return ok
}

func NewUserContext(userID uint, userName, tenantID string) *UserContext {
	return &UserContext{
		UserID:   userID,
		UserName: userName,
		TenantID: tenantID,
	}
}

func (u *UserContext) HasTenant() bool {
	return u != nil && u.TenantID != ""
}

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
