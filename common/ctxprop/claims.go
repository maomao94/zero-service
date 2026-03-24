package ctxprop

import (
	"context"
	"fmt"

	"zero-service/common/ctxdata"
)

// ExtractFromClaims 从 JWT claims map 提取用户字段，注入到 context values。
// 用于用户侧 JWT 认证：claims key 与 ctxdata.PropFields[*].CtxKey 一致（如 "user-id"）。
// JWT 解析后数值类型为 float64，此函数自动转为 string。
func ExtractFromClaims(ctx context.Context, claims map[string]any) context.Context {
	if len(claims) == 0 {
		return ctx
	}
	for _, f := range ctxdata.PropFields {
		if v := ClaimString(claims, f.CtxKey); v != "" {
			ctx = context.WithValue(ctx, f.CtxKey, v)
		}
	}
	return ctx
}

// ClaimString 从 claims map 中提取指定 key 的字符串值。
// 自动处理 JWT 常见类型：string 直接返回，float64（JSON number）转为整数字符串。
func ClaimString(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
