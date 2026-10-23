package user

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/user"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 小程序登录
func MiniProgramLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MiniProgramLoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := user.NewMiniProgramLoginLogic(r.Context(), svcCtx)
		resp, err := l.MiniProgramLogin(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
