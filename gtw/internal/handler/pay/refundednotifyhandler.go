package pay

import (
	"net/http"

	xhttp "github.com/zeromicro/x/http"
	"zero-service/gtw/internal/logic/pay"
	"zero-service/gtw/internal/svc"
)

// 微信退款通知
func RefundedNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := pay.NewRefundedNotifyLogic(r.Context(), svcCtx, r, w)
		err := l.RefundedNotify()
		if err != nil {
			// 例外：微信回调按微信协议返回，成功响应由 PowerWeChat 写出，不走统一信封。
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			//xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
