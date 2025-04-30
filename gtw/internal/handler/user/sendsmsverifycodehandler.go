package user

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
	"zero-service/gtw/internal/logic/user"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 发送手机号验证码
func SendSMSVerifyCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendSMSVerifyCodeRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewSendSMSVerifyCodeLogic(r.Context(), svcCtx)
		resp, err := l.SendSMSVerifyCode(&req)
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
