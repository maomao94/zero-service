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

// ApplyClaimMapping 将外部 JWT claim key 映射为内部标准 key。
// mapping 格式：internalKey -> externalKey（如 "user-id" -> "user_id"）。
// 映射结果直接写入 claims map（覆盖已有值），原始 key 保留。
func ApplyClaimMapping(claims map[string]any, mapping map[string]string) {
	for internalKey, externalKey := range mapping {
		if v, ok := claims[externalKey]; ok {
			claims[internalKey] = v
		}
	}
}

// ApplyClaimMappingToCtx 从 context 中读取外部 claim key 的值，
// 以内部标准 key 重新写入 context。
// 适用于 go-zero WithJwt 场景：JWT claims 已由框架注入 context.Value，
// 此函数将外部 key 的值复制到内部 key，使下游 ctxdata.GetUserId 等可正常工作。
// mapping 格式：internalKey -> externalKey（如 "user-id" -> "user_id"）。
func ApplyClaimMappingToCtx(ctx context.Context, mapping map[string]string) context.Context {
	for internalKey, externalKey := range mapping {
		if v := ctx.Value(externalKey); v != nil {
			ctx = context.WithValue(ctx, internalKey, v)
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
