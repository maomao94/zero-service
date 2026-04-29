package gormx

import "context"

type contextKey string

const (
	userContextKey  contextKey = "gormx:user"
	DefaultTenantID            = "default"
)

type AuditUserID interface {
	~uint | ~uint64 | ~int64 | ~string
}

type UserContext struct {
	UserID   any    `json:"user_id"`
	UserName string `json:"user_name"`
	TenantID string `json:"tenant_id"`
}

func WithUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	if userCtx == nil {
		return ctx
	}
	return context.WithValue(ctx, userContextKey, userCtx)
}

func GetUserContext(ctx context.Context) *UserContext {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(userContextKey).(*UserContext); ok {
		return v
	}
	return nil
}

func GetUserID(ctx context.Context) uint {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return 0
	}
	switch v := userCtx.UserID.(type) {
	case uint:
		return v
	case uint64:
		return uint(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint(v)
	case string:
		return 0
	default:
		return 0
	}
}

func GetUserIDAs[T AuditUserID](ctx context.Context) (T, bool) {
	var zero T
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return zero, false
	}
	v, ok := userCtx.UserID.(T)
	if !ok {
		return zero, false
	}
	return v, true
}

func GetUserIDText(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	if v, ok := userCtx.UserID.(string); ok {
		return v
	}
	return ""
}

func GetUserName(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return ""
	}
	return userCtx.UserName
}

func GetTenantID(ctx context.Context) string {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return DefaultTenantID
	}
	if userCtx.TenantID == "" {
		return DefaultTenantID
	}
	return userCtx.TenantID
}

func NewUserContext[T AuditUserID](userID T, userName, tenantID string) *UserContext {
	return &UserContext{
		UserID:   userID,
		UserName: userName,
		TenantID: tenantID,
	}
}

func NewStringUserContext(userID, userName, tenantID string) *UserContext {
	return NewUserContext(userID, userName, tenantID)
}

func (u *UserContext) AuditUserValue() any {
	if u == nil {
		return nil
	}
	switch v := u.UserID.(type) {
	case uint:
		if v == 0 {
			return nil
		}
		return v
	case uint64:
		if v == 0 {
			return nil
		}
		return v
	case int64:
		if v == 0 {
			return nil
		}
		return v
	case string:
		if v == "" {
			return nil
		}
		return v
	default:
		return nil
	}
}
