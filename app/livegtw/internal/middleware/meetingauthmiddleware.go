// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"net/http"
	"strings"

	"zero-service/common/authctx"
	"zero-service/common/gtwx"
	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
)

type MeetingAuthMiddleware struct {
	claimMapping map[string]string
}

func NewMeetingAuthMiddleware(claimMapping map[string]string) *MeetingAuthMiddleware {
	return &MeetingAuthMiddleware{claimMapping: claimMapping}
}

func (m *MeetingAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if auth := r.Header.Get("Authorization"); auth != "" {
			ctx = authctx.WithAuthorization(ctx, auth)
		}
		ctx = authctx.BridgeJWTClaims(ctx, m.claimMapping)
		if strings.TrimSpace(authctx.GetUserId(ctx)) == "" {
			// 鉴权失败同样返回 HTTP 200 信封，业务结果以 code 为准。
			gtwx.WriteError(ctx, w, xerrors.New(
				int(extproto.Code__1_00_UNAUTHORIZED), "缺少用户身份"))
			return
		}
		next(w, r.WithContext(ctx))
	}
}
