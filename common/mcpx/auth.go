package mcpx

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"
	"zero-service/common/tool"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewDualTokenVerifier 创建双模式 Token 验证器。
// 优先常量时间比较 serviceToken（连接级/服务侧鉴权），
// 失败则尝试 JWT 解析（调用级/用户侧鉴权，UserID 从 claims 提取）。
// TokenInfo.Extra[ctxdata.CtxAuthTypeKey] 标识认证来源："service" 或 "user"。
// claimMapping 支持将外部 JWT claim key 映射为内部标准 key（如 "user-id" -> "user_id"）。
func NewDualTokenVerifier(jwtSecrets []string, serviceToken string, claimMapping map[string]string) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		// 1. ServiceToken 常量时间比较
		if serviceToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(serviceToken)) == 1 {
			logx.WithContext(ctx).Debugf("[mcpx-auth] service token matched")
			return &auth.TokenInfo{
				Expiration: time.Now().Add(24 * time.Hour),
				Extra:      map[string]any{ctxdata.CtxAuthTypeKey: "service"},
			}, nil
		}

		// 2. JWT 解析（用户侧认证）
		if len(jwtSecrets) > 0 {
			claims, err := tool.ParseToken(token, jwtSecrets...)
			if err != nil {
				logx.WithContext(ctx).Debugf("[mcpx-auth] jwt parse failed: %v", err)
				return nil, auth.ErrInvalidToken
			}

			// 应用外部 claim key 映射（如 "user_id" -> "user-id"）
			ctxprop.ApplyClaimMapping(claims, claimMapping)

			// 构建 Extra：只收集 PropFields + exp（供 CallToolWrapper 提取）
			extra := make(map[string]any, len(ctxdata.PropFields)+2)
			extra[ctxdata.CtxAuthTypeKey] = "user"
			for _, f := range ctxdata.PropFields {
				if v, ok := claims[f.CtxKey]; ok {
					extra[f.CtxKey] = v
				}
			}
			if v, ok := claims["exp"]; ok {
				extra["exp"] = v
			}

			info := &auth.TokenInfo{
				UserID: ctxprop.ClaimString(claims, ctxdata.CtxUserIdKey),
				Extra:  extra,
			}
			extra[ctxdata.CtxAuthorizationKey] = token
			if exp, ok := claims["exp"].(float64); ok {
				info.Expiration = time.Unix(int64(exp), 0)
			}
			logx.WithContext(ctx).Debugf("[mcpx-auth] jwt verified, userId=%s, extra=%v", info.UserID, extra)
			return info, nil
		}

		logx.WithContext(ctx).Debugf("[mcpx-auth] no verifier matched")
		return nil, auth.ErrInvalidToken
	}
}
