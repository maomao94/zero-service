package authctx

import (
	"context"
)

const (
	CtxUserIdKey        = "user-id"
	CtxUserNameKey      = "user-name"
	CtxDeptCodeKey      = "dept-code"
	CtxAuthorizationKey = "authorization"
	CtxAuthTypeKey      = "auth-type"
	CtxDeviceIdKey      = "device-id"
)

// ContextKeys lists authentication context keys in propagation order.
// These string constants remain the wire/claim naming table (JWT claims,
// gRPC metadata, MCP _meta) and are not used for process context reads/writes.
var ContextKeys = []string{
	CtxAuthorizationKey,
	CtxUserIdKey,
	CtxUserNameKey,
	CtxDeptCodeKey,
	CtxAuthTypeKey,
	CtxDeviceIdKey,
}

// DefaultClaimMapping maps standard internal keys to common external JWT claim names.
// This allows automatic parsing of underscore-style claims (user_id, user_name, dept_code)
// without requiring explicit configuration.
var DefaultClaimMapping = map[string]string{
	CtxUserIdKey:   "user_id",
	CtxUserNameKey: "user_name",
	CtxDeptCodeKey: "dept_code",
}

// Package-private typed keys hold process-context identity values.
type userIDKey struct{}
type userNameKey struct{}
type deptCodeKey struct{}
type authorizationKey struct{}
type authTypeKey struct{}
type deviceIDKey struct{}

// WithUserID stores v under the typed user-id context key.
func WithUserID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, userIDKey{}, v)
}

// WithUserName stores v under the typed user-name context key.
func WithUserName(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, userNameKey{}, v)
}

// WithDeptCode stores v under the typed dept-code context key.
func WithDeptCode(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, deptCodeKey{}, v)
}

// WithAuthorization stores v under the typed authorization context key.
func WithAuthorization(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, authorizationKey{}, v)
}

// WithAuthType stores v under the typed auth-type context key.
func WithAuthType(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, authTypeKey{}, v)
}

// WithDeviceID stores v under the typed device-id context key.
func WithDeviceID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, deviceIDKey{}, v)
}

// WithKey stores v under the typed key matching the wire/claim key.
// Unknown keys are ignored.
func WithKey(ctx context.Context, key, v string) context.Context {
	switch key {
	case CtxUserIdKey:
		return WithUserID(ctx, v)
	case CtxUserNameKey:
		return WithUserName(ctx, v)
	case CtxDeptCodeKey:
		return WithDeptCode(ctx, v)
	case CtxAuthorizationKey:
		return WithAuthorization(ctx, v)
	case CtxAuthTypeKey:
		return WithAuthType(ctx, v)
	case CtxDeviceIdKey:
		return WithDeviceID(ctx, v)
	default:
		return ctx
	}
}

// GetUserId reads the typed user-id context key. It returns "" when absent
// or when the stored value is not a string.
func GetUserId(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey{}).(string); ok {
		return v
	}
	return ""
}

// GetUserName reads the typed user-name context key.
func GetUserName(ctx context.Context) string {
	if v, ok := ctx.Value(userNameKey{}).(string); ok {
		return v
	}
	return ""
}

// GetAuthorization reads the typed authorization context key.
func GetAuthorization(ctx context.Context) string {
	if v, ok := ctx.Value(authorizationKey{}).(string); ok {
		return v
	}
	return ""
}

// GetDeptCode reads the typed dept-code context key.
func GetDeptCode(ctx context.Context) string {
	if v, ok := ctx.Value(deptCodeKey{}).(string); ok {
		return v
	}
	return ""
}

// GetAuthType reads the typed auth-type context key.
func GetAuthType(ctx context.Context) string {
	if v, ok := ctx.Value(authTypeKey{}).(string); ok {
		return v
	}
	return ""
}

// GetDeviceId reads the typed device-id context key.
func GetDeviceId(ctx context.Context) string {
	if v, ok := ctx.Value(deviceIDKey{}).(string); ok {
		return v
	}
	return ""
}

// GetByKey reads the value stored under the typed key matching the wire/claim key.
// It returns "" for unknown keys or non-string values.
func GetByKey(ctx context.Context, key string) string {
	switch key {
	case CtxUserIdKey:
		return GetUserId(ctx)
	case CtxUserNameKey:
		return GetUserName(ctx)
	case CtxDeptCodeKey:
		return GetDeptCode(ctx)
	case CtxAuthorizationKey:
		return GetAuthorization(ctx)
	case CtxAuthTypeKey:
		return GetAuthType(ctx)
	case CtxDeviceIdKey:
		return GetDeviceId(ctx)
	default:
		return ""
	}
}

// BridgeJWTClaims converts JWT claims written by go-zero's Authorize middleware
// (as raw string context keys) into typed context keys. It MUST run after JWT
// verification in the request chain (register in gateway server.Use).
//
// It first copies dash-named standard wire keys (user-id/user-name/dept-code/…),
// then applies ClaimMapping so underscore external names (user_id/…) map onto the
// standard typed keys. user-id may arrive as int64, float64, or string and is
// normalized to its string form; other values (bool/array/map/…) are skipped,
// and an existing typed value is never overwritten.
//
// When mapping is nil, DefaultClaimMapping is used to automatically parse
// common underscore-style claims (user_id, user_name, dept_code).
func BridgeJWTClaims(ctx context.Context, mapping map[string]string) context.Context {
	if mapping == nil {
		mapping = DefaultClaimMapping
	}
	for _, key := range ContextKeys {
		if v := toStringClaim(ctx.Value(key)); v != "" && GetByKey(ctx, key) == "" {
			ctx = WithKey(ctx, key, v)
		}
	}
	for targetKey, sourceKey := range mapping {
		if v := toStringClaim(ctx.Value(sourceKey)); v != "" && GetByKey(ctx, targetKey) == "" {
			ctx = WithKey(ctx, targetKey, v)
		}
	}
	return ctx
}

// toStringClaim converts a claim value to its safe string form.
func toStringClaim(v any) string {
	return normalizeClaimString(v)
}
