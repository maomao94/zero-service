// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"net/http"

	"zero-service/common/authctx"
)

type UserAuthMiddleware struct {
	claimMapping map[string]string
}

func NewUserAuthMiddleware(claimMapping map[string]string) *UserAuthMiddleware {
	return &UserAuthMiddleware{claimMapping: claimMapping}
}

func (m *UserAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if auth := r.Header.Get("Authorization"); auth != "" {
			ctx = authctx.WithAuthorization(ctx, auth)
		}
		ctx = authctx.BridgeJWTClaims(ctx, m.claimMapping)
		next(w, r.WithContext(ctx))
	}
}
