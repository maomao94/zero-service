package mcpx

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"zero-service/common/tool"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewDualTokenVerifier 创建双模式 Token 验证器。
// 优先常量时间比较 serviceToken（连接级鉴权），
// 失败则尝试 JWT 解析（调用级鉴权）。
// UserID 始终为空，因为 mcpx 是多用户共享 session。
func NewDualTokenVerifier(jwtSecrets []string, serviceToken string) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		// 1. ServiceToken 常量时间比较
		if serviceToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(serviceToken)) == 1 {
			logx.Debugf("[mcpx-auth] service token matched")
			return &auth.TokenInfo{
				Expiration: time.Now().Add(24 * time.Hour),
				Extra:      map[string]any{"type": "service"},
			}, nil
		}

		// 2. JWT 解析
		if len(jwtSecrets) > 0 {
			claims, err := tool.ParseToken(token, jwtSecrets...)
			if err != nil {
				logx.Debugf("[mcpx-auth] jwt parse failed: %v", err)
				return nil, auth.ErrInvalidToken
			}
			info := &auth.TokenInfo{Extra: map[string]any(claims)}
			if exp, ok := claims["exp"].(float64); ok {
				info.Expiration = time.Unix(int64(exp), 0)
			}
			logx.Debugf("[mcpx-auth] jwt verified, claims keys=%v", mapKeys(claims))
			return info, nil
		}

		logx.Debugf("[mcpx-auth] no verifier matched")
		return nil, auth.ErrInvalidToken
	}
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:12] + "..."
}

// mapKeys 提取 map 的所有 key。
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
