package mcpx

import (
	"context"
	"crypto/subtle"
	"net/http"
	"sort"
	"time"

	"zero-service/common/authctx"
	"zero-service/common/tool"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewDualTokenVerifier 创建双模式 Token 验证器。
// 优先常量时间比较 serviceToken（连接级/服务侧鉴权），
// 失败则尝试 JWT 解析（调用级/用户侧鉴权，UserID 从 claims 提取）。
// TokenInfo.Extra[authctx.CtxAuthTypeKey] 标识认证来源："service" 或 "user"。
// claimMapping 使用 internalKey -> externalKey，将外部 JWT claim 映射为内部标准 key
// （如 "user-id" -> "user_id"）。
func NewDualTokenVerifier(jwtSecrets []string, serviceToken string, claimMapping map[string]string) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		// 1. ServiceToken 常量时间比较
		if serviceToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(serviceToken)) == 1 {
			logx.WithContext(ctx).Debugf("[mcpx-auth] service token matched")
			return &auth.TokenInfo{
				UserID:     "service",
				Expiration: time.Now().Add(24 * time.Hour),
				Extra:      map[string]any{authctx.CtxAuthTypeKey: "service"},
			}, nil
		}

		// 2. JWT 解析（用户侧认证）
		if len(jwtSecrets) > 0 {
			claims, err := tool.ParseToken(token, jwtSecrets...)
			if err != nil {
				logx.WithContext(ctx).Debugf("[mcpx-auth] jwt parse failed: %v", err)
				return nil, auth.ErrInvalidToken
			}

			// 将外部 JWT claim 映射到内部标准 key，方向为 internalKey -> externalKey。
			authctx.ApplyClaimMapping(claims, claimMapping)

			// 构建 Extra：只收集标准认证 context keys 和 exp，供 CallToolWrapper 提取。
			extra := make(map[string]any, len(authctx.ContextKeys)+2)
			extra[authctx.CtxAuthTypeKey] = "user"
			for _, key := range authctx.ContextKeys {
				if v, ok := claims[key]; ok {
					extra[key] = v
				}
			}
			if v, ok := claims["exp"]; ok {
				extra["exp"] = v
			}

			info := &auth.TokenInfo{
				UserID: authctx.ClaimString(claims, authctx.CtxUserIdKey),
				Extra:  extra,
			}
			extra[authctx.CtxAuthorizationKey] = token
			if exp, ok := claims["exp"].(float64); ok {
				info.Expiration = time.Unix(int64(exp), 0)
			}
			logx.WithContext(ctx).Debugf("[mcpx-auth] jwt verified, userId=%s, extraKeys=%v", info.UserID, mapKeys(extra))
			return info, nil
		}

		logx.WithContext(ctx).Error("[mcpx] no verifier matched")
		return nil, auth.ErrInvalidToken
	}
}

// mapKeys returns the sorted key names of m. Values are never exposed.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
