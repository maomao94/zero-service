package handler

import (
	"net/http"

	"zero-service/common/authctx"
)

// MeetingAuthMiddleware 业务 API 路由组中间件。
// 在 rest.WithJwt 验证之后运行，设置 auth-type、保存 Authorization、桥接 JWT claims。
// webhook / 测试页等不挂此中间件。
type MeetingAuthMiddleware struct {
	MeetingAuth func(next http.HandlerFunc) http.HandlerFunc
}

func NewMeetingAuthMiddleware(claimMapping map[string]string) *MeetingAuthMiddleware {
	return &MeetingAuthMiddleware{
		MeetingAuth: func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				if auth := r.Header.Get("Authorization"); auth != "" {
					ctx = authctx.WithAuthType(ctx, "user")
					ctx = authctx.WithAuthorization(ctx, auth)
				}
				ctx = authctx.BridgeJWTClaims(ctx, claimMapping)
				next(w, r.WithContext(ctx))
			}
		},
	}
}
